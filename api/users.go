package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/fatih/structs"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"

	jwt "github.com/dgrijalva/jwt-go"
	jwtRequest "github.com/dgrijalva/jwt-go/request"
	log "github.com/sirupsen/logrus"
)

const (
	saltSize = 1 << 5
	hashCost = bcrypt.DefaultCost + 3

	privKeyPath = "app.rsa"
	pubKeyPath  = "app.rsa.pub"

	userRetrievalQuery = `SELECT * FROM users WHERE email = $1`
	userExistenceQuery = `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND archived_on IS NULL)`
	userDeletionQuery  = `UPDATE users SET archived_on = NOW() WHERE email = $1 AND archived_on IS NULL`
)

var (
	verifyKey *rsa.PublicKey
	signKey   *rsa.PrivateKey
)

// This is a lame hack function that exists solely for tests because im bad at programming
func mustLoadKey(err error, l Fataler) {
	if err != nil {
		l.Fatal(err)
	}
}

func init() {
	fatalLogger := log.New()

	signBytes, err := ioutil.ReadFile(privKeyPath)
	mustLoadKey(err, fatalLogger)
	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	mustLoadKey(err, fatalLogger)

	verifyBytes, err := ioutil.ReadFile(pubKeyPath)
	mustLoadKey(err, fatalLogger)
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	mustLoadKey(err, fatalLogger)
}

// User represents a Dairycart user
type User struct {
	DBRow
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Salt      []byte `json:"salt"`
	IsAdmin   bool   `json:"is_admin"`
}

// DisplayUser represents a Dairycart user we can return in responses
type DisplayUser struct {
	DBRow
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	IsAdmin   bool   `json:"is_admin"`
}

// UserCreationInput represents the payload used to create a Dairycart user
type UserCreationInput struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,gte=64"`
	IsAdmin   bool   `json:"is_admin"`
}

// UserLoginInput represents the payload used to log in a Dairycart user
type UserLoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse represents what we return to the user
type TokenResponse struct {
	Token string `json:"token"`
}

func validateTokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		token, err := jwtRequest.ParseFromRequest(req, jwtRequest.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				return verifyKey, nil
			})
		if err == nil && token.Valid {
			next(res, req)
		} else {
			log.Printf("received the following error with a token: %s", err.Error())
			res.WriteHeader(http.StatusUnauthorized)
			errRes := &ErrorResponse{
				Status:  http.StatusUnauthorized,
				Message: "Unauthorized access to this resource",
			}
			json.NewEncoder(res).Encode(errRes)
		}
	}
}

func createUserFromInput(in *UserCreationInput) (*User, error) {
	salt, err := generateSalt()
	if err != nil {
		return nil, err
	}

	saltedAndHashedPassword, err := saltAndHashPassword(in.Password, salt)
	if err != nil {
		return nil, err
	}

	user := &User{
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Email:     in.Email,
		Password:  string(saltedAndHashedPassword),
		Salt:      salt,
		IsAdmin:   in.IsAdmin,
	}
	return user, nil
}

func generateSalt() ([]byte, error) {
	b := make([]byte, saltSize)
	_, err := rand.Read(b)
	return b, err
}

func saltAndHashPassword(password string, salt []byte) (string, error) {
	passwordToHash := append(salt, password...)
	saltedAndHashedPassword, err := bcrypt.GenerateFromPassword(passwordToHash, hashCost)
	return string(saltedAndHashedPassword), err
}

func validateUserCreationInput(req *http.Request) (*UserCreationInput, error) {
	newUser := &UserCreationInput{}
	err := json.NewDecoder(req.Body).Decode(newUser)
	if err != nil {
		return nil, err
	}

	p := structs.New(newUser)
	// go will happily decode an invalid input into a completely zeroed struct,
	// so we gotta do checks like this because we're bad at programming.
	if p.IsZero() {
		return nil, errors.New("Invalid input provided for user body")
	}

	validate := validator.New()
	err = validate.Struct(newUser)
	if err != nil {
		return nil, err
	}

	return newUser, err
}

func createUserInDB(db *sqlx.DB, u *User) (uint64, error) {
	var newUserID uint64
	query, args := buildUserCreationQuery(u)
	err := db.QueryRow(query, args...).Scan(&newUserID)
	return newUserID, err
}

func buildUserCreationHandler(db *sqlx.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		userInput, err := validateUserCreationInput(req)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}

		// can't create a user with an email that already exists!
		exists, err := rowExistsInDB(db, userExistenceQuery, userInput.Email)
		if err != nil || exists {
			notifyOfInvalidRequestBody(res, fmt.Errorf("user with email `%s` already exists", userInput.Email))
			return
		}

		newUser, err := createUserFromInput(userInput)
		if err != nil {
			notifyOfInternalIssue(res, err, "creating user")
			return
		}

		createdUserID, err := createUserInDB(db, newUser)
		if err != nil {
			notifyOfInternalIssue(res, err, "insert user in database")
			return
		}
		responseUser := &DisplayUser{
			DBRow: DBRow{
				ID:        createdUserID,
				CreatedOn: time.Now(),
			},
			FirstName: newUser.FirstName,
			LastName:  newUser.LastName,
			Email:     newUser.Email,
			IsAdmin:   newUser.IsAdmin,
		}

		res.WriteHeader(http.StatusCreated)
		json.NewEncoder(res).Encode(responseUser)
	}
}

func retrieveUserFromDB(db *sqlx.DB, email string) (User, error) {
	var u User
	err := db.Get(&u, userRetrievalQuery, email)
	return u, err
}

func passwordIsValid(in *UserLoginInput, u User) bool {
	saltedInputPassword := append(u.Salt, in.Password...)
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), saltedInputPassword)
	return err == nil
}

func validateLoginInput(req *http.Request) (*UserLoginInput, error) {
	loginInfo := &UserLoginInput{}
	err := json.NewDecoder(req.Body).Decode(loginInfo)
	if err != nil {
		return nil, err
	}

	i := structs.New(loginInfo)
	// go will happily decode an invalid input into a completely zeroed struct,
	// so we gotta do checks like this because we're bad at programming.
	if i.IsZero() {
		return nil, errors.New("Invalid input provided for user login")
	}

	return loginInfo, nil
}

func buildToken() (TokenResponse, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.MapClaims{
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(7 * (24 * time.Hour)).Unix(),
	})
	tokenString, err := token.SignedString(signKey)
	tr := TokenResponse{
		Token: tokenString,
	}
	return tr, err
}

func buildUserLoginHandler(db *sqlx.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		loginInput, err := validateLoginInput(req)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}

		user, err := retrieveUserFromDB(db, loginInput.Email)
		if err != nil {
			notifyOfInternalIssue(res, err, "retrieve user")
			return
		}

		loginValid := passwordIsValid(loginInput, user)
		if !loginValid {
			notifyOfInvalidAuthenticationAttempt(res, loginInput.Email)
			return
		}

		jsonWebToken, err := buildToken()
		if err != nil {
			notifyOfInternalIssue(res, err, "generate token")
			return
		}

		statusToWrite := http.StatusUnauthorized
		if loginValid {
			statusToWrite = http.StatusOK
		}

		res.WriteHeader(statusToWrite)
		json.NewEncoder(res).Encode(jsonWebToken)
	}
}

func archiveUser(db *sqlx.DB, email string) error {
	_, err := db.Exec(userDeletionQuery, email)
	return err
}