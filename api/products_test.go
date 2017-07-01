package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

const (
	badSKUUpdateJSON = `{"sku": "pooƃ ou sᴉ nʞs sᴉɥʇ"}`
	lolFloats        = 12.34000015258789

	exampleProductUpdateInput = `
		{
			"sku": "example",
			"name": "Test",
			"quantity": 666,
			"upc": "1234567890",
			"price": 12.34
		}
	`
	exampleTimeAvailableString = "2016-12-31T12:00:00Z"
)

var (
	productHeaders        []string
	exampleProductData    []driver.Value
	exampleProduct        *Product
	exampleUpdatedProduct *Product
)

func init() {
	exampleProduct = &Product{
		DBRow: DBRow{
			ID:        2,
			CreatedOn: generateExampleTimeForTests(),
		},
		SKU:  "skateboard",
		Name: "Skateboard",
		// Subtitle:      NullString{sql.NullString{String: "", Valid: true}},
		UPC: NullString{sql.NullString{String: "1234567890", Valid: true}},
		// Manufacturer:  NullString{sql.NullString{String: "", Valid: true}},
		// Brand:         NullString{sql.NullString{String: "", Valid: true}},
		Quantity:      123,
		Price:         99.99,
		Cost:          50.00,
		Description:   "This is a skateboard. Please wear a helmet.",
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		AvailableOn:   generateExampleTimeForTests(),
	}
	exampleProduct.Subtitle.Valid = true
	exampleProduct.Manufacturer.Valid = true
	exampleProduct.Brand.Valid = true

	productHeaders = strings.Split(strings.TrimSpace(productTableHeaders), ",\n\t\t")
	exampleProductData = []driver.Value{
		exampleProduct.ID,
		exampleProduct.Name,
		exampleProduct.Subtitle.String,
		exampleProduct.Description,
		exampleProduct.SKU,
		exampleProduct.UPC.String,
		exampleProduct.Manufacturer.String,
		exampleProduct.Brand.String,
		exampleProduct.Quantity,
		exampleProduct.Taxable,
		exampleProduct.Price,
		exampleProduct.OnSale,
		exampleProduct.SalePrice,
		exampleProduct.Cost,
		exampleProduct.ProductWeight,
		exampleProduct.ProductHeight,
		exampleProduct.ProductWidth,
		exampleProduct.ProductLength,
		exampleProduct.PackageWeight,
		exampleProduct.PackageHeight,
		exampleProduct.PackageWidth,
		exampleProduct.PackageLength,
		exampleProduct.QuantityPerPackage,
		exampleProduct.AvailableOn,
		exampleProduct.CreatedOn,
		nil,
		nil,
	}

	exampleUpdatedProduct = &Product{
		DBRow: DBRow{
			ID:        exampleProduct.ID,
			CreatedOn: generateExampleTimeForTests(),
		},
		SKU:      "example",
		Name:     "Test",
		UPC:      NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity: 666,
		Cost:     50.00,
		Price:    lolFloats,
	}

}

func setExpectationsForProductExistence(mock sqlmock.Sqlmock, SKU string, exists bool, err error) {
	exampleRows := sqlmock.NewRows([]string{""}).AddRow(strconv.FormatBool(exists))
	mock.ExpectQuery(formatQueryForSQLMock(skuExistenceQuery)).
		WithArgs(SKU).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductExistenceByID(mock sqlmock.Sqlmock, productID string, exists bool, err error) {
	exampleRows := sqlmock.NewRows([]string{""}).AddRow(strconv.FormatBool(exists))
	mock.ExpectQuery(formatQueryForSQLMock(productExistenceQuery)).
		WithArgs(productID).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductListQuery(mock sqlmock.Sqlmock, err error) {
	exampleRows := sqlmock.NewRows(productHeaders).
		AddRow(exampleProductData...).
		AddRow(exampleProductData...).
		AddRow(exampleProductData...)

	allProductsRetrievalQuery, _ := buildProductListQuery(defaultQueryFilter)
	mock.ExpectQuery(formatQueryForSQLMock(allProductsRetrievalQuery)).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductRetrieval(mock sqlmock.Sqlmock, sku string, err error) {
	exampleRows := sqlmock.NewRows(productHeaders).AddRow(exampleProductData...)
	skuRetrievalQuery := formatQueryForSQLMock(completeProductRetrievalQuery)
	mock.ExpectQuery(skuRetrievalQuery).
		WithArgs(sku).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductUpdate(mock sqlmock.Sqlmock, p *Product, err error) {
	exampleRows := sqlmock.NewRows(productHeaders).AddRow(exampleProductData...)
	productUpdateQuery, queryArgs := buildProductUpdateQuery(p)
	args := argsToDriverValues(queryArgs)
	mock.ExpectQuery(formatQueryForSQLMock(productUpdateQuery)).
		WithArgs(args...).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductCreation(mock sqlmock.Sqlmock, p *Product, err error) {
	exampleRows := sqlmock.NewRows([]string{"id"}).AddRow(p.ID)
	productCreationQuery, args := buildProductCreationQuery(p)
	queryArgs := argsToDriverValues(args)
	mock.ExpectQuery(formatQueryForSQLMock(productCreationQuery)).
		WithArgs(queryArgs...).
		WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductUpdateHandler(mock sqlmock.Sqlmock, p *Product, err error) {
	exampleRows := sqlmock.NewRows(productHeaders).AddRow(exampleProductData...)
	productUpdateQuery, _ := buildProductUpdateQuery(exampleProduct)
	mock.ExpectQuery(formatQueryForSQLMock(productUpdateQuery)).
		WithArgs(
			p.Cost,
			p.Name,
			p.Price,
			p.Quantity,
			p.SKU,
			p.UPC.String,
			p.ID,
		).WillReturnRows(exampleRows).
		WillReturnError(err)
}

func setExpectationsForProductDeletion(mock sqlmock.Sqlmock, sku string) {
	mock.ExpectExec(formatQueryForSQLMock(productDeletionQuery)).
		WithArgs(sku).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestValidateProductUpdateInputWithValidInput(t *testing.T) {
	t.Parallel()
	expected := &Product{
		SKU:      exampleSKU,
		Name:     "Test",
		UPC:      NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity: 666,
		Price:    12.34,
		Cost:     0,
	}

	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(exampleProductUpdateInput))
	actual, err := validateProductUpdateInput(req)

	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "valid product input should parse into a proper product struct")
}

func TestValidateProductUpdateInputWithInvalidInput(t *testing.T) {
	t.Parallel()
	exampleInput := strings.NewReader(`{"testing": true}`)

	req := httptest.NewRequest("GET", "http://example.com", exampleInput)
	_, err := validateProductUpdateInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductUpdateInputWithCompletelyInvalidInput(t *testing.T) {
	t.Parallel()
	exampleInput := strings.NewReader(`{"testing":}`)

	req := httptest.NewRequest("GET", "http://example.com", exampleInput)
	_, err := validateProductUpdateInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductUpdateInputWithInvalidSKU(t *testing.T) {
	t.Parallel()
	exampleInput := strings.NewReader(badSKUUpdateJSON)

	req := httptest.NewRequest("GET", "http://example.com", exampleInput)
	_, err := validateProductUpdateInput(req)

	assert.NotNil(t, err)
}

func TestRetrieveProductFromDB(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	setExpectationsForProductRetrieval(mock, exampleSKU, nil)

	actual, err := retrieveProductFromDB(db, exampleSKU)
	assert.Nil(t, err)
	assert.Equal(t, *exampleProduct, actual, "expected and actual products should match")
	ensureExpectationsWereMet(t, mock)
}

func TestRetrieveProductFromDBWhenDBReturnsError(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	setExpectationsForProductRetrieval(mock, exampleSKU, sql.ErrNoRows)

	_, err := retrieveProductFromDB(db, exampleSKU)
	assert.NotNil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestDeleteProductBySKU(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	setExpectationsForProductDeletion(mock, exampleSKU)

	err := deleteProductBySKU(db, exampleSKU)
	assert.Nil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestUpdateProductInDatabase(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	setExpectationsForProductUpdate(mock, exampleProduct, nil)

	err := updateProductInDatabase(db, exampleProduct)
	assert.Nil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestCreateProductInDB(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	mock.ExpectBegin()
	setExpectationsForProductCreation(mock, exampleProduct, nil)
	mock.ExpectCommit()

	tx, err := db.Begin()
	assert.Nil(t, err)

	newID, err := createProductInDB(tx, exampleProduct)
	assert.Nil(t, err)
	assert.Equal(t, exampleProduct.ID, newID, "createProductInDB should return the created ID")

	err = tx.Commit()
	assert.Nil(t, err)
	ensureExpectationsWereMet(t, mock)
}

func TestValidateProductCreationInput(t *testing.T) {
	t.Parallel()
	exampleProductCreationInput := `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1
		}
	`
	expected := &ProductCreationInput{
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		SKU:           "skateboard",
		Name:          "Skateboard",
		UPC:           "1234567890",
		Quantity:      123,
		Price:         12.34,
		Cost:          5,
	}

	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(exampleProductCreationInput))
	actual, err := validateProductCreationInput(req)

	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "valid product input should parse into a proper product struct")
}

func TestValidateProductCreationInputWithEmptyInput(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "http://example.com", nil)
	_, err := validateProductCreationInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductCreationInputWithInvalidInput(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(exampleGarbageInput))
	_, err := validateProductCreationInput(req)

	assert.NotNil(t, err)
}

func TestValidateProductCreationInputWithInvalidSKU(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "http://example.com", strings.NewReader(badSKUUpdateJSON))
	_, err := validateProductCreationInput(req)

	assert.NotNil(t, err)
}

////////////////////////////////////////////////////////
//                                                    //
//                 HTTP Handler Tests                 //
//                                                    //
////////////////////////////////////////////////////////

func TestProductExistenceHandlerWithExistingProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)

	req, err := http.NewRequest("HEAD", "/v1/product/example", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductExistenceHandlerWithNonexistentProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "unreal", false, nil)

	req, err := http.NewRequest("HEAD", "/v1/product/unreal", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductExistenceHandlerWithExistenceCheckerError(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "unreal", false, arbitraryError)

	req, err := http.NewRequest("HEAD", "/v1/product/unreal", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductRetrievalHandlerWithExistingProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductRetrieval(mock, exampleProduct.SKU, nil)

	req, err := http.NewRequest("GET", fmt.Sprintf("/v1/product/%s", exampleProduct.SKU), nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductRetrievalHandlerWithDBError(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductRetrieval(mock, exampleProduct.SKU, arbitraryError)

	req, err := http.NewRequest("GET", fmt.Sprintf("/v1/product/%s", exampleProduct.SKU), nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductRetrievalHandlerWithNonexistentProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductRetrieval(mock, exampleProduct.SKU, sql.ErrNoRows)

	req, err := http.NewRequest("GET", fmt.Sprintf("/v1/product/%s", exampleProduct.SKU), nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductListHandler(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForRowCount(mock, "products", defaultQueryFilter, 3, nil)
	setExpectationsForProductListQuery(mock, nil)

	req, err := http.NewRequest("GET", "/v1/products", nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 200, res.Code, "status code should be 200")

	expected := &ProductsResponse{
		ListResponse: ListResponse{
			Page:  1,
			Limit: 25,
			Count: 3,
		},
	}

	actual := &ProductsResponse{}
	err = json.NewDecoder(strings.NewReader(res.Body.String())).Decode(actual)
	assert.Nil(t, err)

	assert.Equal(t, expected.Page, actual.Page, "expected and actual product pages should be equal")
	assert.Equal(t, expected.Limit, actual.Limit, "expected and actual product limits should be equal")
	assert.Equal(t, expected.Count, actual.Count, "expected and actual product counts should be equal")
	assert.Equal(t, uint64(len(actual.Data)), actual.Count, "actual product counts and product response count field should be equal")
	ensureExpectationsWereMet(t, mock)
}

func TestProductListHandlerWithErrorRetrievingCount(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForRowCount(mock, "products", defaultQueryFilter, 3, arbitraryError)

	req, err := http.NewRequest("GET", "/v1/products", nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 500, res.Code, "status code should be 500")

	ensureExpectationsWereMet(t, mock)
}

func TestProductListHandlerWithDBError(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForRowCount(mock, "products", defaultQueryFilter, 3, nil)
	setExpectationsForProductListQuery(mock, arbitraryError)

	req, err := http.NewRequest("GET", "/v1/products", nil)
	assert.Nil(t, err)

	router.ServeHTTP(res, req)
	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandler(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductRetrieval(mock, exampleProduct.SKU, nil)
	setExpectationsForProductUpdateHandler(mock, exampleUpdatedProduct, nil)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/v1/product/%s", exampleProduct.SKU), strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithNonexistentProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductRetrieval(mock, exampleProduct.SKU, sql.ErrNoRows)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/v1/product/%s", exampleProduct.SKU), strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithInputValidationError(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)

	req, err := http.NewRequest("PUT", "/v1/product/example", strings.NewReader(badSKUUpdateJSON))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Code, "status code should be 400")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithDBErrorRetrievingProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductRetrieval(mock, exampleProduct.SKU, arbitraryError)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/v1/product/%s", exampleProduct.SKU), strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductUpdateHandlerWithDBErrorUpdatingProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductRetrieval(mock, exampleProduct.SKU, nil)
	setExpectationsForProductUpdateHandler(mock, exampleUpdatedProduct, arbitraryError)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/v1/product/%s", exampleProduct.SKU), strings.NewReader(exampleProductUpdateInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductDeletionHandlerWithExistentProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, true, nil)
	setExpectationsForProductDeletion(mock, exampleSKU)

	req, err := http.NewRequest("DELETE", "/v1/product/example", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 200, res.Code, "status code should be 200")
	ensureExpectationsWereMet(t, mock)
}

func TestProductDeletionHandlerWithNonexistentProduct(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, exampleSKU, false, nil)

	req, err := http.NewRequest("DELETE", "/v1/product/example", nil)
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 404, res.Code, "status code should be 404")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandler(t *testing.T) {
	t.Parallel()
	exampleProductCreationInputWithOptions := `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1,
			"options": [{
				"name": "something",
				"values": [
					"one",
					"two",
					"three"
				]
			}]
		}
	`
	expectedProduct := &Product{
		DBRow: DBRow{
			ID:        2,
			CreatedOn: generateExampleTimeForTests(),
		},
		Name:          "Skateboard",
		SKU:           "skateboard",
		Price:         lolFloats,
		Cost:          5,
		Taxable:       true,
		Description:   "This is a skateboard. Please wear a helmet.",
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		UPC:           NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:      123,
	}

	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin()
	setExpectationsForProductCreation(mock, expectedProduct, nil)
	setExpectationsForProductOptionCreation(mock, expectedCreatedProductOption, exampleProduct.ID, nil)
	setExpectationsForProductOptionValueCreation(mock, &expectedCreatedProductOption.Values[0], nil)
	setExpectationsForProductOptionValueCreation(mock, &expectedCreatedProductOption.Values[1], nil)
	setExpectationsForProductOptionValueCreation(mock, &expectedCreatedProductOption.Values[2], nil)
	mock.ExpectCommit()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithOptions))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 201, res.Code, "status code should be 201")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWhereCommitReturnsAnError(t *testing.T) {
	t.Parallel()
	exampleProductCreationInputWithOptions := `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1,
			"options": [{
				"name": "something",
				"values": [
					"one",
					"two",
					"three"
				]
			}]
		}
	`
	expectedProduct := &Product{
		DBRow: DBRow{
			ID:        2,
			CreatedOn: generateExampleTimeForTests(),
		},
		Name:          "Skateboard",
		SKU:           "skateboard",
		Price:         lolFloats,
		Cost:          5,
		Taxable:       true,
		Description:   "This is a skateboard. Please wear a helmet.",
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		UPC:           NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:      123,
	}

	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin()
	setExpectationsForProductCreation(mock, expectedProduct, nil)
	setExpectationsForProductOptionCreation(mock, expectedCreatedProductOption, exampleProduct.ID, nil)
	setExpectationsForProductOptionValueCreation(mock, &expectedCreatedProductOption.Values[0], nil)
	setExpectationsForProductOptionValueCreation(mock, &expectedCreatedProductOption.Values[1], nil)
	setExpectationsForProductOptionValueCreation(mock, &expectedCreatedProductOption.Values[2], nil)
	mock.ExpectCommit().WillReturnError(arbitraryError)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithOptions))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWhereTransactionFailsToBegin(t *testing.T) {
	t.Parallel()
	exampleProductCreationInputWithOptions := `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1,
			"options": [{
				"name": "something",
				"values": [
					"one",
					"two",
					"three"
				]
			}]
		}
	`
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin().WillReturnError(arbitraryError)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithOptions))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWithoutOptions(t *testing.T) {
	t.Parallel()
	exampleProductCreationInput := `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1
		}
	`
	expectedProduct := &Product{
		DBRow: DBRow{
			ID:        2,
			CreatedOn: generateExampleTimeForTests(),
		},
		Name:          "Skateboard",
		SKU:           "skateboard",
		Price:         lolFloats,
		Cost:          5,
		Taxable:       true,
		Description:   "This is a skateboard. Please wear a helmet.",
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		UPC:           NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:      123,
	}

	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)

	mock.ExpectBegin()
	setExpectationsForProductCreation(mock, expectedProduct, nil)
	mock.ExpectCommit()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 201, res.Code, "status code should be 201")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWithInvalidProductInput(t *testing.T) {
	t.Parallel()
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(badSKUUpdateJSON))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Code, "status code should be 400")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerForAlreadyExistentProduct(t *testing.T) {
	t.Parallel()
	exampleProductCreationInput := `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1
		}
	`
	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", true, nil)

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 400, res.Code, "status code should be 400")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWithErrorCreatingOptions(t *testing.T) {
	t.Parallel()
	exampleProductCreationInputWithOptions := fmt.Sprintf(`
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 99.99,
			"cost": 50,
			"description": "This is a skateboard. Please wear a helmet.",
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1,
			"available_on": "%s",
			"options": [{
				"name": "something",
				"values": [
					"one",
					"two",
					"three"
				]
			}]
		}
	`, exampleTimeAvailableString)

	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)
	mock.ExpectBegin()
	setExpectationsForProductCreation(mock, exampleProduct, nil)
	setExpectationsForProductOptionCreation(mock, expectedCreatedProductOption, exampleProduct.ID, arbitraryError)
	mock.ExpectRollback()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInputWithOptions))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}

func TestProductCreationHandlerWhereProductCreationFails(t *testing.T) {
	t.Parallel()
	exampleProductCreationInput := `
		{
			"sku": "skateboard",
			"name": "Skateboard",
			"upc": "1234567890",
			"quantity": 123,
			"price": 12.34,
			"cost": 5,
			"description": "This is a skateboard. Please wear a helmet.",
			"taxable": true,
			"product_weight": 8,
			"product_height": 7,
			"product_width": 6,
			"product_length": 5,
			"package_weight": 4,
			"package_height": 3,
			"package_width": 2,
			"package_length": 1
		}
	`
	expectedProduct := &Product{
		DBRow: DBRow{
			ID:        2,
			CreatedOn: generateExampleTimeForTests(),
		},
		Name:          "Skateboard",
		SKU:           "skateboard",
		Price:         lolFloats,
		Cost:          5,
		Description:   "This is a skateboard. Please wear a helmet.",
		Taxable:       true,
		ProductWeight: 8,
		ProductHeight: 7,
		ProductWidth:  6,
		ProductLength: 5,
		PackageWeight: 4,
		PackageHeight: 3,
		PackageWidth:  2,
		PackageLength: 1,
		UPC:           NullString{sql.NullString{String: "1234567890", Valid: true}},
		Quantity:      123,
	}

	db, mock := setupDBForTest(t)
	res, router := setupMockRequestsAndMux(db)
	setExpectationsForProductExistence(mock, "skateboard", false, nil)
	mock.ExpectBegin()
	setExpectationsForProductCreation(mock, expectedProduct, arbitraryError)
	mock.ExpectRollback()

	req, err := http.NewRequest("POST", "/v1/product", strings.NewReader(exampleProductCreationInput))
	assert.Nil(t, err)
	router.ServeHTTP(res, req)

	assert.Equal(t, 500, res.Code, "status code should be 500")
	ensureExpectationsWereMet(t, mock)
}
