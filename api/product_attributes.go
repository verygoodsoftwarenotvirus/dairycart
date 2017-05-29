package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/fatih/structs"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

// ProductAttribute represents a products variant attributes. If you have a t-shirt that comes in three colors
// and three sizes, then there are two ProductAttributes for that base_product, color and size.
type ProductAttribute struct {
	ID                  int64                    `json:"id"`
	Name                string                   `json:"name"`
	ProductProgenitorID int64                    `json:"product_progenitor_id"`
	Values              []*ProductAttributeValue `json:"values"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           pq.NullTime              `json:"-"`
	ArchivedAt          pq.NullTime              `json:"-"`
}

func (a *ProductAttribute) generateScanArgs() []interface{} {
	return []interface{}{
		&a.ID,
		&a.Name,
		&a.ProductProgenitorID,
		&a.CreatedAt,
		&a.UpdatedAt,
		&a.ArchivedAt,
	}
}

// ProductAttributesResponse is a product attribute response struct
type ProductAttributesResponse struct {
	ListResponse
	Data []ProductAttribute `json:"data"`
}

// ProductAttributeUpdateInput is a struct to use for updating product attributes
type ProductAttributeUpdateInput struct {
	Name string `json:"name"`
}

// ProductAttributeCreationInput is a struct to use for creating product attributes
type ProductAttributeCreationInput struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// retrieveProductAttributeFromDB retrieves a ProductAttribute with a given ID from the database
func retrieveProductAttributeFromDB(db *sql.DB, id int64) (*ProductAttribute, error) {
	attribute := &ProductAttribute{}
	scanArgs := attribute.generateScanArgs()
	query := buildProductAttributeRetrievalQuery(id)
	err := db.QueryRow(query, id).Scan(scanArgs...)
	if err == sql.ErrNoRows {
		return attribute, errors.Wrap(err, "Error querying for product")
	}
	return attribute, err
}

func getProductAttributesForProgenitor(db *sql.DB, progenitorID string, queryFilter *QueryFilter) ([]ProductAttribute, error) {
	var attributes []ProductAttribute

	rows, err := db.Query(buildProductAttributeListQuery(progenitorID, queryFilter))
	if err != nil {
		return nil, errors.Wrap(err, "Error encountered querying for products")
	}
	defer rows.Close()
	for rows.Next() {
		var attribute ProductAttribute
		_ = rows.Scan(attribute.generateScanArgs()...)
		attributes = append(attributes, attribute)
	}
	return attributes, nil
}

func buildProductAttributeListHandler(db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		progenitorID := mux.Vars(req)["progenitor_id"]
		rawFilterParams := req.URL.Query()
		queryFilter := parseRawFilterParams(rawFilterParams)
		attributes, err := getProductAttributesForProgenitor(db, progenitorID, queryFilter)
		if err != nil {
			notifyOfInternalIssue(res, err, "retrieve products from the database")
			return
		}

		attributesResponse := &ProductAttributesResponse{
			ListResponse: ListResponse{
				Page:  queryFilter.Page,
				Limit: queryFilter.Limit,
				Count: uint64(len(attributes)),
			},
			Data: attributes,
		}
		json.NewEncoder(res).Encode(attributesResponse)
	}
}

func validateProductAttributeUpdateInput(req *http.Request) (*ProductAttributeUpdateInput, error) {
	i := &ProductAttributeUpdateInput{}
	json.NewDecoder(req.Body).Decode(i)

	s := structs.New(i)
	// go will happily decode an invalid input into a completely zeroed struct,
	// so we gotta do checks like this because we're bad at programming.
	if s.IsZero() {
		return nil, errors.New("Invalid input provided for product attribute body")
	}

	return i, nil
}

func updateProductAttributeInDB(db *sql.DB, a *ProductAttribute) error {
	productUpdateQuery, queryArgs := buildProductAttributeUpdateQuery(a)
	row := db.QueryRow(productUpdateQuery, queryArgs...)
	scanArgs := a.generateScanArgs()
	err := row.Scan(scanArgs...)
	//err := db.QueryRow(productUpdateQuery, queryArgs...).Scan(a.generateScanArgs()...)
	return err
}

func buildProductAttributeUpdateHandler(db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// ProductUpdateHandler is a request handler that can update products
		reqVars := mux.Vars(req)
		progenitorID := reqVars["progenitor_id"]
		attributeID := reqVars["attribute_id"]
		// eating this error because Mux should validate this for us.
		attributeIDInt, _ := strconv.Atoi(attributeID)

		noop := func(i interface{}) {
			return
		}

		// can't update an attribute for a product progenitor that doesn't exist!
		progenitorExists, err := rowExistsInDB(db, "product_progenitors", "id", progenitorID)
		if err != nil || !progenitorExists {
			respondThatRowDoesNotExist(req, res, "product progenitor", progenitorID)
			return
		}

		// can't update an attribute that doesn't exist!
		attributeExists, err := rowExistsInDB(db, "product_attributes", "id", attributeID)
		if err != nil || !attributeExists {
			respondThatRowDoesNotExist(req, res, "product attribute", attributeID)
			return
		}

		updatedAttributeData, err := validateProductAttributeUpdateInput(req)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}

		existingAttribute, err := retrieveProductAttributeFromDB(db, int64(attributeIDInt))
		if err != nil {
			errStr := err.Error()
			noop(errStr)
			notifyOfInternalIssue(res, err, "retrieve product attribute from the database")
			return
		}
		existingAttribute.Name = updatedAttributeData.Name

		err = updateProductAttributeInDB(db, existingAttribute)
		if err != nil {
			errStr := err.Error()
			noop(errStr)
			notifyOfInternalIssue(res, err, "update product attribute in the database")
			return
		}

		json.NewEncoder(res).Encode(existingAttribute)

	}
}

func validateProductAttributeCreationInput(req *http.Request) (*ProductAttributeCreationInput, error) {
	i := &ProductAttributeCreationInput{}
	json.NewDecoder(req.Body).Decode(i)

	s := structs.New(i)
	// go will happily decode an invalid input into a completely zeroed struct,
	// so we gotta do checks like this because we're bad at programming.
	if s.IsZero() {
		return nil, errors.New("Invalid input provided for product attribute body")
	}

	return i, nil
}

func createProductAttributeInDB(db *sql.DB, a *ProductAttribute) (*ProductAttribute, error) {
	var newAttributeID int64
	// using QueryRow instead of Exec because we want it to return the newly created row's ID
	// Exec normally returns a sql.Result, which has a LastInsertedID() method, but when I tested
	// this locally, it never worked. ¯\_(ツ)_/¯
	query, queryArgs := buildProductAttributeCreationQuery(a)
	err := db.QueryRow(query, queryArgs...).Scan(&newAttributeID)

	a.ID = newAttributeID
	return a, err
}

func createProductAttributeAndValuesInDBFromInput(db *sql.DB, in *ProductAttributeCreationInput, progenitorID int64) (*ProductAttribute, error) {
	newProductAttribute := &ProductAttribute{
		Name:                in.Name,
		ProductProgenitorID: progenitorID,
	}
	newProductAttribute, err := createProductAttributeInDB(db, newProductAttribute)
	if err != nil {
		return nil, err
	}

	for _, value := range in.Values {
		newAttributeValue := &ProductAttributeValue{
			ProductAttributeID: newProductAttribute.ID,
			Value:              value,
		}
		newAttributeValueID, err := createProductAttributeValueInDB(db, newAttributeValue)
		if err != nil {
			return nil, err
		}
		newAttributeValue.ID = newAttributeValueID
		newProductAttribute.Values = append(newProductAttribute.Values, newAttributeValue)
	}

	return newProductAttribute, nil
}

func buildProductAttributeCreationHandler(db *sql.DB) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// ProductUpdateHandler is a request handler that can update products
		progenitorID := mux.Vars(req)["progenitor_id"]
		// eating this error because Mux should valdiate this for us.
		progenitorIDInt, _ := strconv.Atoi(progenitorID)

		// can't update an attribute for a product progenitor that doesn't exist!
		exists, err := rowExistsInDB(db, "product_progenitors", "id", progenitorID)
		if err != nil || !exists {
			respondThatRowDoesNotExist(req, res, "product progenitor", progenitorID)
			return
		}

		// TODO: check that an existing attribute with that name doesn't exist

		newAttributeData, err := validateProductAttributeCreationInput(req)
		if err != nil {
			notifyOfInvalidRequestBody(res, err)
			return
		}

		newProductAttribute, err := createProductAttributeAndValuesInDBFromInput(db, newAttributeData, int64(progenitorIDInt))
		if err != nil {
			notifyOfInternalIssue(res, err, "retrieve products from the database")
			return
		}

		json.NewEncoder(res).Encode(newProductAttribute)
	}
}
