package dairytest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	// "text/template"

	"github.com/dairycart/dairycart/api/storage/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func replaceTimeStringsForProductTests(body string) string {
	re := regexp.MustCompile(`(?U)(,?)"(available_on)":"(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,6})?Z)?"(,?)`)
	return strings.TrimSpace(re.ReplaceAllString(body, ""))
}

func logBodyAndResetResponse(t *testing.T, resp *http.Response) {
	t.Helper()
	respStr := turnResponseBodyIntoString(t, resp)
	log.Printf(`

		%s

	`, respStr)
	resp.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(respStr)))
}

func createTestProduct(t *testing.T, p models.ProductCreationInput) uint64 {
	t.Helper()
	newProductJSON := createJSONBody(t, p)
	resp, err := createProduct(newProductJSON)
	require.Nil(t, err)
	assertStatusCode(t, resp, http.StatusCreated)

	var res models.Product
	unmarshalBody(t, resp, &res)
	return res.ID
}

func deleteTestProduct(t *testing.T, sku string) {
	resp, err := deleteProduct(sku)
	require.Nil(t, err)
	assertStatusCode(t, resp, http.StatusOK)
}

func deleteTestProductRoot(t *testing.T, productRootID uint64) {
	resp, err := deleteProductRoot(strconv.Itoa(int(productRootID)))
	require.Nil(t, err)
	assertStatusCode(t, resp, http.StatusOK)
}

func compareListResponses(t *testing.T, expected, actual models.ListResponse) {
	t.Helper()
	assert.Equal(t, expected.Limit, actual.Limit, "expected and actual limit don't match")
	assert.Equal(t, expected.Page, actual.Page, "expected and actual page don't match")
}

func createJSONBody(t *testing.T, o interface{}) string {
	t.Helper()
	b, err := json.Marshal(o)
	require.Nil(t, err)
	str := string(b)
	return str
}

func TestProductExistenceRoute(t *testing.T) {
	// t.Parallel()

	t.Run("for existing product", func(*testing.T) {
		resp, err := checkProductExistence(existentSKU)
		require.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		actual := turnResponseBodyIntoString(t, resp)
		assert.Equal(t, "", actual, "product existence body should be empty")
	})

	t.Run("for nonexistent product", func(*testing.T) {
		resp, err := checkProductExistence(nonexistentSKU)
		require.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		actual := turnResponseBodyIntoString(t, resp)
		assert.Equal(t, "", actual, "product existence body for nonexistent product should be empty")
	})
}

// TODO: maybe these functions should just set the values that we don't care about equality for rather than check for the equality of each field
// for instance, we don't really worry about IDs, so make this function set the expected.ID to actual.ID and then use assert to check equality
func compareProducts(t *testing.T, expected models.Product, actual models.Product) {
	t.Helper()
	// assert.Equal(t, expected.ProductRootID, actual.ProductRootID, "expected and actual ProductRootID values don't match")
	assert.Equal(t, expected.Name, actual.Name, "expected and actual Name values don't match")
	assert.Equal(t, expected.Subtitle, actual.Subtitle, "expected and actual Subtitle values don't match")
	assert.Equal(t, expected.Description, actual.Description, "expected and actual Description values don't match")
	assert.Equal(t, expected.OptionSummary, actual.OptionSummary, "expected and actual OptionSummary values don't match")
	assert.Equal(t, expected.SKU, actual.SKU, "expected and actual SKU values don't match")
	assert.Equal(t, expected.UPC, actual.UPC, "expected and actual UPC values don't match")
	assert.Equal(t, expected.Manufacturer, actual.Manufacturer, "expected and actual Manufacturer values don't match")
	assert.Equal(t, expected.Brand, actual.Brand, "expected and actual Brand values don't match")
	assert.Equal(t, expected.Quantity, actual.Quantity, "expected and actual Quantity values don't match")
	assert.Equal(t, expected.Taxable, actual.Taxable, "expected and actual Taxable values don't match")
	assert.Equal(t, expected.Price, actual.Price, "expected and actual Price values don't match")
	assert.Equal(t, expected.OnSale, actual.OnSale, "expected and actual OnSale values don't match")
	assert.Equal(t, expected.SalePrice, actual.SalePrice, "expected and actual SalePrice values don't match")
	assert.Equal(t, expected.Cost, actual.Cost, "expected and actual Cost values don't match")
	assert.Equal(t, expected.ProductWeight, actual.ProductWeight, "expected and actual ProductWeight values don't match")
	assert.Equal(t, expected.ProductHeight, actual.ProductHeight, "expected and actual ProductHeight values don't match")
	assert.Equal(t, expected.ProductWidth, actual.ProductWidth, "expected and actual ProductWidth values don't match")
	assert.Equal(t, expected.ProductLength, actual.ProductLength, "expected and actual ProductLength values don't match")
	assert.Equal(t, expected.PackageWeight, actual.PackageWeight, "expected and actual PackageWeight values don't match")
	assert.Equal(t, expected.PackageHeight, actual.PackageHeight, "expected and actual PackageHeight values don't match")
	assert.Equal(t, expected.PackageWidth, actual.PackageWidth, "expected and actual PackageWidth values don't match")
	assert.Equal(t, expected.PackageLength, actual.PackageLength, "expected and actual PackageLength values don't match")
	assert.Equal(t, expected.QuantityPerPackage, actual.QuantityPerPackage, "expected and actual QuantityPerPackage values don't match")

	for i := range expected.ApplicableOptionValues {
		if len(actual.ApplicableOptionValues)-1 < i {
			t.Logf("expected %d option values attached to product, got %d instead.", len(expected.ApplicableOptionValues), len(actual.ApplicableOptionValues))
			t.Fail()
			break
		}
		compareProductOptionValues(t, expected.ApplicableOptionValues[i], actual.ApplicableOptionValues[i])
	}
}

func compareProductOptionValues(t *testing.T, expected, actual models.ProductOptionValue) {
	assert.Equal(t, expected.Value, actual.Value, "expected and actual Value should match")
}

func compareProductOptions(t *testing.T, expected, actual models.ProductOption) {
	assert.Equal(t, expected.Name, actual.Name, "expected and actual Name should match")
	for i := range expected.Values {
		if len(actual.Values)-1 < i {
			t.Logf("expected %d option values, got %d instead.", len(expected.Values), len(actual.Values))
			t.Fail()
			break
		}
		compareProductOptionValues(t, expected.Values[i], actual.Values[i])
	}
}

func compareProductRoots(t *testing.T, expected, actual models.ProductRoot) {
	t.Helper()
	assert.Equal(t, expected.Name, actual.Name, "expected and actual Name should match")
	assert.Equal(t, expected.Subtitle, actual.Subtitle, "expected and actual Subtitle should match")
	assert.Equal(t, expected.Description, actual.Description, "expected and actual Description should match")
	assert.Equal(t, expected.SKUPrefix, actual.SKUPrefix, "expected and actual SKUPrefix should match")
	assert.Equal(t, expected.Manufacturer, actual.Manufacturer, "expected and actual Manufacturer should match")
	assert.Equal(t, expected.Brand, actual.Brand, "expected and actual Brand should match")
	assert.Equal(t, expected.Taxable, actual.Taxable, "expected and actual Taxable should match")
	assert.Equal(t, expected.Cost, actual.Cost, "expected and actual Cost should match")
	assert.Equal(t, expected.ProductWeight, actual.ProductWeight, "expected and actual ProductWeight should match")
	assert.Equal(t, expected.ProductHeight, actual.ProductHeight, "expected and actual ProductHeight should match")
	assert.Equal(t, expected.ProductWidth, actual.ProductWidth, "expected and actual ProductWidth should match")
	assert.Equal(t, expected.ProductLength, actual.ProductLength, "expected and actual ProductLength should match")
	assert.Equal(t, expected.PackageWeight, actual.PackageWeight, "expected and actual PackageWeight should match")
	assert.Equal(t, expected.PackageHeight, actual.PackageHeight, "expected and actual PackageHeight should match")
	assert.Equal(t, expected.PackageWidth, actual.PackageWidth, "expected and actual PackageWidth should match")
	assert.Equal(t, expected.PackageLength, actual.PackageLength, "expected and actual PackageLength should match")
	assert.Equal(t, expected.QuantityPerPackage, actual.QuantityPerPackage, "expected and actual QuantityPerPackage should match")

	for i := range expected.Options {
		if len(actual.Options)-1 < i {
			t.Logf("expected %d options, got %d instead.", len(expected.Options), len(actual.Options))
			t.Fail()
			break
		}
		compareProductOptions(t, expected.Options[i], actual.Options[i])
	}

	for i := range expected.Products {
		if len(actual.Products)-1 < i {
			t.Logf("expected %d products, got %d instead.", len(expected.Products), len(actual.Products))
			t.Fail()
			break
		}
		compareProducts(t, expected.Products[i], actual.Products[i])
	}
}

func TestProductRetrievalRoute(t *testing.T) {
	// t.Parallel()
	t.Run("existent product", func(*testing.T) {
		resp, err := retrieveProduct(existentSKU)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		expected := models.Product{
			ProductRootID:          1,
			Name:                   "Your Favorite Band's T-Shirt",
			Subtitle:               "A t-shirt you can wear",
			Description:            "Wear this if you'd like. Or don't, I'm not in charge of your actions",
			OptionSummary:          "Size: Small, Color: Red",
			SKU:                    existentSKU,
			Manufacturer:           "Record Company",
			Brand:                  "Your Favorite Band",
			Quantity:               666,
			Taxable:                true,
			Price:                  20,
			OnSale:                 false,
			SalePrice:              0,
			Cost:                   10,
			ProductWeight:          1,
			ProductHeight:          5,
			ProductWidth:           5,
			ProductLength:          5,
			PackageWeight:          1,
			PackageHeight:          5,
			PackageWidth:           5,
			PackageLength:          5,
			QuantityPerPackage:     1,
			ApplicableOptionValues: nil,
		}

		var actual models.Product
		unmarshalBody(t, resp, &actual)
		compareProducts(t, expected, actual)
	})

	t.Run("nonexistent product", func(*testing.T) {
		resp, err := retrieveProduct(nonexistentSKU)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		expected := models.ErrorResponse{
			Status:  http.StatusNotFound,
			Message: fmt.Sprintf("The product you were looking for (sku '%s') does not exist", nonexistentSKU),
		}
		assert.Equal(t, expected, actual, "trying to retrieve a product that doesn't exist should respond 404")
	})
}

func TestProductListRoute(t *testing.T) {
	// // t.Parallel()

	t.Run("with standard filter", func(*testing.T) {
		resp, err := retrieveListOfProducts(nil)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		expected := models.ListResponse{
			Limit: 25,
			Page:  1,
		}
		var actual models.ListResponse
		unmarshalBody(t, resp, &actual)
		compareListResponses(t, expected, actual)
	})

	// FIXME
	// t.Run("with nonstandard filter", func(*testing.T) {
	// 	customFilter := map[string]string{
	// 		"page":  "2",
	// 		"limit": "5",
	// 	}
	// 	resp, err := retrieveListOfProducts(customFilter)
	// 	assert.Nil(t, err)
	// 	assertStatusCode(t, resp, http.StatusOK)

	// 	expected := models.ListResponse{
	// 		Limit: 5,
	// 		Page:  2,
	// 	}
	// 	var actual models.ListResponse
	// 	unmarshalBody(t, resp, &actual)
	// 	compareListResponses(t, expected, actual)
	// })
}

func TestProductUpdateRoute(t *testing.T) {
	// // t.Parallel()
	testSKU := "test-product-updating"

	t.Run("normal use", func(*testing.T) {
		testProduct := models.ProductCreationInput{
			Name:               "New Product",
			Subtitle:           "this is a product",
			Description:        "this product is neat or maybe its not who really knows for sure?",
			SKU:                testSKU,
			Manufacturer:       "Manufacturer",
			Brand:              "Brand",
			Quantity:           123,
			QuantityPerPackage: 3,
			Taxable:            false,
			Price:              12.34,
			OnSale:             true,
			SalePrice:          10,
			Cost:               5,
			ProductWeight:      9,
			ProductHeight:      9,
			ProductWidth:       9,
			ProductLength:      9,
			PackageWeight:      9,
			PackageHeight:      9,
			PackageWidth:       9,
			PackageLength:      9,
		}
		productRootID := createTestProduct(t, testProduct)
		JSONBody := `{"quantity":666}`
		resp, err := updateProduct(testSKU, JSONBody)
		require.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		expected := models.Product{
			ProductRootID:      productRootID,
			Name:               "New Product",
			Subtitle:           "this is a product",
			Description:        "this product is neat or maybe its not who really knows for sure?",
			SKU:                testSKU,
			Manufacturer:       "Manufacturer",
			Brand:              "Brand",
			Quantity:           666,
			QuantityPerPackage: 3,
			Taxable:            false,
			Price:              12.34,
			OnSale:             true,
			SalePrice:          10,
			Cost:               5,
			ProductWeight:      9,
			ProductHeight:      9,
			ProductWidth:       9,
			ProductLength:      9,
			PackageWeight:      9,
			PackageHeight:      9,
			PackageWidth:       9,
			PackageLength:      9,
		}
		var actual models.Product
		unmarshalBody(t, resp, &actual)
		compareProducts(t, expected, actual)
		deleteTestProduct(t, testSKU)
	})

	t.Run("with invalid input", func(*testing.T) {
		resp, err := updateProduct(existentSKU, exampleGarbageInput)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusBadRequest)

		actual := turnResponseBodyIntoString(t, resp)
		assert.Equal(t, expectedBadRequestResponse, actual, "product update route should respond with failure message when you try to update a product with invalid input")
	})

	t.Run("with invalid sku", func(*testing.T) {
		JSONBody := `{"sku": "thí% $kü ïs not åny gõôd"}`
		resp, err := updateProduct(existentSKU, JSONBody)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusBadRequest)
	})

	t.Run("for nonexistent product", func(*testing.T) {
		JSONBody := `{"quantity":666}`
		resp, err := updateProduct(nonexistentSKU, JSONBody)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		actual := turnResponseBodyIntoString(t, resp)
		expected := minifyJSON(t, `
			{
				"status": 404,
				"message": "The product you were looking for (sku 'nonexistent') does not exist"
			}
		`)
		assert.Equal(t, expected, actual, "trying to update a product that doesn't exist should respond 404")
	})
}

func TestProductCreationRoute(t *testing.T) {
	testSKU := "test-product-creation"

	t.Run("normal usage", func(*testing.T) {
		testProduct := models.ProductCreationInput{
			Name:               "New Product",
			Subtitle:           "this is a product",
			Description:        "this product is neat or maybe its not who really knows for sure?",
			OptionSummary:      "",
			SKU:                testSKU,
			UPC:                "0123456789",
			Manufacturer:       "Manufacturer",
			Brand:              "Brand",
			Quantity:           123,
			Taxable:            false,
			Price:              12.34,
			OnSale:             true,
			SalePrice:          10,
			Cost:               5,
			ProductWeight:      9,
			ProductHeight:      9,
			ProductWidth:       9,
			ProductLength:      9,
			PackageWeight:      9,
			PackageHeight:      9,
			PackageWidth:       9,
			PackageLength:      9,
			QuantityPerPackage: 3,
		}

		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var actual models.ProductRoot
		unmarshalBody(t, resp, &actual)

		expected := models.ProductRoot{
			Name:               "New Product",
			Subtitle:           "this is a product",
			Description:        "this product is neat or maybe its not who really knows for sure?",
			SKUPrefix:          testSKU,
			Manufacturer:       "Manufacturer",
			Brand:              "Brand",
			Taxable:            false,
			Cost:               5,
			ProductWeight:      9,
			ProductHeight:      9,
			ProductWidth:       9,
			ProductLength:      9,
			PackageWeight:      9,
			PackageHeight:      9,
			PackageWidth:       9,
			PackageLength:      9,
			QuantityPerPackage: 3,
			Options:            []models.ProductOption{},
			Products:           []models.Product{convertCreationInputToProduct(testProduct)},
		}

		compareProductRoots(t, expected, actual)
		deleteTestProductRoot(t, actual.ID)
	})

	t.Run("with options", func(*testing.T) {
		testProduct := models.ProductCreationInput{
			Name:               "New Product",
			Subtitle:           "this is a product",
			Description:        "this product is neat or maybe its not who really knows for sure?",
			OptionSummary:      "",
			SKU:                testSKU,
			Manufacturer:       "Manufacturer",
			Brand:              "Brand",
			Quantity:           123,
			Taxable:            false,
			Price:              12.34,
			OnSale:             true,
			SalePrice:          10,
			Cost:               5,
			ProductWeight:      9,
			ProductHeight:      9,
			ProductWidth:       9,
			ProductLength:      9,
			PackageWeight:      9,
			PackageHeight:      9,
			PackageWidth:       9,
			PackageLength:      9,
			QuantityPerPackage: 3,
			Options:            []models.ProductOptionCreationInput{{Name: "numbers", Values: []string{"one", "two", "three"}}},
		}

		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var actual models.ProductRoot
		unmarshalBody(t, resp, &actual)

		expected := models.ProductRoot{
			Name:               "New Product",
			Subtitle:           "this is a product",
			Description:        "this product is neat or maybe its not who really knows for sure?",
			SKUPrefix:          testSKU,
			Manufacturer:       "Manufacturer",
			Brand:              "Brand",
			Taxable:            false,
			Cost:               5,
			ProductWeight:      9,
			ProductHeight:      9,
			ProductWidth:       9,
			ProductLength:      9,
			PackageWeight:      9,
			PackageHeight:      9,
			PackageWidth:       9,
			PackageLength:      9,
			QuantityPerPackage: 3,
			Options: []models.ProductOption{
				{
					Name:   "numbers",
					Values: []models.ProductOptionValue{{Value: "one"}, {Value: "two"}, {Value: "three"}},
				},
			},
			Products: []models.Product{
				{
					Name:                   "New Product",
					Subtitle:               "this is a product",
					Description:            "this product is neat or maybe its not who really knows for sure?",
					OptionSummary:          "numbers: one",
					SKU:                    fmt.Sprintf("%s_%s", testSKU, "one"),
					Manufacturer:           "Manufacturer",
					Brand:                  "Brand",
					Quantity:               123,
					Taxable:                false,
					Price:                  12.34,
					OnSale:                 true,
					SalePrice:              10,
					Cost:                   5,
					ProductWeight:          9,
					ProductHeight:          9,
					ProductWidth:           9,
					ProductLength:          9,
					PackageWeight:          9,
					PackageHeight:          9,
					PackageWidth:           9,
					PackageLength:          9,
					QuantityPerPackage:     3,
					ApplicableOptionValues: []models.ProductOptionValue{{Value: "one"}},
				},
				{
					Name:                   "New Product",
					Subtitle:               "this is a product",
					Description:            "this product is neat or maybe its not who really knows for sure?",
					OptionSummary:          "numbers: two",
					SKU:                    fmt.Sprintf("%s_%s", testSKU, "two"),
					Manufacturer:           "Manufacturer",
					Brand:                  "Brand",
					Quantity:               123,
					Taxable:                false,
					Price:                  12.34,
					OnSale:                 true,
					SalePrice:              10,
					Cost:                   5,
					ProductWeight:          9,
					ProductHeight:          9,
					ProductWidth:           9,
					ProductLength:          9,
					PackageWeight:          9,
					PackageHeight:          9,
					PackageWidth:           9,
					PackageLength:          9,
					QuantityPerPackage:     3,
					ApplicableOptionValues: []models.ProductOptionValue{{Value: "two"}},
				},
				{
					Name:                   "New Product",
					Subtitle:               "this is a product",
					Description:            "this product is neat or maybe its not who really knows for sure?",
					OptionSummary:          "numbers: three",
					SKU:                    fmt.Sprintf("%s_%s", testSKU, "three"),
					Manufacturer:           "Manufacturer",
					Brand:                  "Brand",
					Quantity:               123,
					Taxable:                false,
					Price:                  12.34,
					OnSale:                 true,
					SalePrice:              10,
					Cost:                   5,
					ProductWeight:          9,
					ProductHeight:          9,
					ProductWidth:           9,
					ProductLength:          9,
					PackageWeight:          9,
					PackageHeight:          9,
					PackageWidth:           9,
					PackageLength:          9,
					QuantityPerPackage:     3,
					ApplicableOptionValues: []models.ProductOptionValue{{Value: "three"}},
				},
			},
		}

		compareProductRoots(t, expected, actual)
	})

	t.Run("with already existent SKU", func(*testing.T) {
		alreadyExistentSKU := "t-shirt"
		testProduct := models.ProductCreationInput{SKU: alreadyExistentSKU}

		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusBadRequest)

		expected := models.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: fmt.Sprintf("product with sku '%s' already exists", alreadyExistentSKU),
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("with invalid input", func(*testing.T) {
		resp, err := createProduct(exampleGarbageInput)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusBadRequest)

		expected := models.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "Invalid input provided in request body",
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})
}

func TestProductDeletion(t *testing.T) {
	// t.Parallel()
	testSKU := "test-product-deletion"

	t.Run("normal usecase", func(*testing.T) {
		testProduct := models.ProductCreationInput{SKU: testSKU}
		createTestProduct(t, testProduct)

		resp, err := deleteProduct(testSKU)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		var actual models.Product
		unmarshalBody(t, resp, &actual)
		assert.False(t, actual.ArchivedOn.Time.IsZero())
	})

	t.Run("nonexistent product", func(*testing.T) {
		resp, err := deleteProduct(nonexistentSKU)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		expected := models.ErrorResponse{
			Status:  http.StatusNotFound,
			Message: fmt.Sprintf("The product you were looking for (sku '%s') does not exist", nonexistentSKU),
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})
}

func TestProductRootList(t *testing.T) {
	// 	t.Parallel()

	t.Run("no filter", func(*testing.T) {
		resp, err := retrieveProductRoots(nil)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		expected := models.ListResponse{
			Limit: 25,
			Page:  1,
		}
		var actual models.ListResponse
		unmarshalBody(t, resp, &actual)
		compareListResponses(t, expected, actual)
	})

	// FIXME
	// t.Run("custom filter", func(*testing.T) {
	// 	customFilter := map[string]string{
	// 		"page":  "2",
	// 		"limit": "1",
	// 	}
	// 	resp, err := retrieveProductRoots(customFilter)
	// 	assert.Nil(t, err)
	// 	assertStatusCode(t, resp, http.StatusOK)

	// 	expected := models.ListResponse{
	// 		Limit: 1,
	// 		Page:  2,
	// 	}
	// 	var actual models.ListResponse
	// 	unmarshalBody(t, resp, &actual)
	// 	compareListResponses(t, expected, actual)
	// })
}

func TestProductRootRetrievalRoute(t *testing.T) {
	// t.Parallel()

	t.Run("normal usage", func(*testing.T) {
		resp, err := retrieveProductRoot(existentID)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		expected := models.ProductRoot{
			Name:               "Your Favorite Band's T-Shirt",
			Subtitle:           "A t-shirt you can wear",
			Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
			SKUPrefix:          "t-shirt",
			Manufacturer:       "Record Company",
			Brand:              "Your Favorite Band",
			Taxable:            true,
			Cost:               20,
			ProductWeight:      1,
			ProductHeight:      5,
			ProductWidth:       5,
			ProductLength:      5,
			PackageWeight:      1,
			PackageHeight:      5,
			PackageWidth:       5,
			PackageLength:      5,
			QuantityPerPackage: 1,
			Options: []models.ProductOption{
				{
					Name: "color",
					// FIXME:
					// Values: []models.ProductOptionValue{{Value: "red"}, {Value: "green"}, {Value: "blue"}}},
				},
				{
					Name: "size",
					// Values: []models.ProductOptionValue{{Value: "small"}, {Value: "medium"}, {Value: "large"}}},
				},
			},
			Products: []models.Product{
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-small-red",
					OptionSummary:      "Size: Small, Color: Red",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// FIXME:
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "small"}, {Value: "red"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-medium-red",
					OptionSummary:      "Size: Medium, Color: Red",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "medium"}, {Value: "red"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-large-red",
					OptionSummary:      "Size: Large, Color: Red",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "large"}, {Value: "red"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-small-blue",
					OptionSummary:      "Size: Small, Color: Blue",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "small"}, {Value: "blue"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-medium-blue",
					OptionSummary:      "Size: Medium, Color: Blue",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "medium"}, {Value: "blue"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-large-blue",
					OptionSummary:      "Size: Large, Color: Blue",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "large"}, {Value: "blue"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-small-green",
					OptionSummary:      "Size: Small, Color: Green",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "small"}, {Value: "green"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-medium-green",
					OptionSummary:      "Size: Medium, Color: Green",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "medium"}, {Value: "green"}},
				},
				{
					Name:               "Your Favorite Band's T-Shirt",
					Subtitle:           "A t-shirt you can wear",
					Description:        "Wear this if you'd like. Or don't, I'm not in charge of your actions",
					SKU:                "t-shirt-large-green",
					OptionSummary:      "Size: Large, Color: Green",
					Manufacturer:       "Record Company",
					Brand:              "Your Favorite Band",
					Taxable:            true,
					Quantity:           666,
					Price:              20,
					Cost:               10,
					ProductWeight:      1,
					ProductHeight:      5,
					ProductWidth:       5,
					ProductLength:      5,
					PackageWeight:      1,
					PackageHeight:      5,
					PackageWidth:       5,
					PackageLength:      5,
					QuantityPerPackage: 1,
					// ApplicableOptionValues: []models.ProductOptionValue{{Value: "large"}, {Value: "green"}},
				},
			},
		}
		var actual models.ProductRoot
		unmarshalBody(t, resp, &actual)
		compareProductRoots(t, expected, actual)
	})

	t.Run("nonexistent product root", func(*testing.T) {
		resp, err := retrieveProductRoot(nonexistentID)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		expected := models.ErrorResponse{
			Status:  http.StatusNotFound,
			Message: fmt.Sprintf("The product_root you were looking for (identified by '%s') does not exist", nonexistentID),
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})
}

func TestProductRootDeletionRoute(t *testing.T) {
	// 	t.Parallel()

	testSKU := "test-product-root-deletion"
	t.Run("normal usage", func(*testing.T) {
		testProduct := models.ProductCreationInput{SKU: testSKU}

		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var createdRoot models.ProductRoot
		unmarshalBody(t, resp, &createdRoot)
		assert.True(t, createdRoot.ArchivedOn.Time.IsZero())

		resp, err = deleteProductRoot(strconv.Itoa(int(createdRoot.ID)))
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		var actual models.ProductRoot
		unmarshalBody(t, resp, &actual)
		assert.True(t, createdRoot.ArchivedOn.Valid)
	})

	t.Run("nonexistent product root", func(*testing.T) {
		resp, err := deleteProductRoot(nonexistentID)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		expected := models.ErrorResponse{
			Status:  http.StatusNotFound,
			Message: fmt.Sprintf("The product_root you were looking for (identified by '%s') does not exist", nonexistentID),
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})
}

func TestProductOptionListRoute(t *testing.T) {
	// 	t.Parallel()

	t.Run("no filter", func(*testing.T) {
		resp, err := retrieveProductOptions("1", nil)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		expected := models.ListResponse{
			Limit: 25,
			Page:  1,
		}
		var actual models.ListResponse
		unmarshalBody(t, resp, &actual)
		compareListResponses(t, expected, actual)
	})

	// FIXME
	// t.Run("custom filter", func(*testing.T) {
	// 	customFilter := map[string]string{
	// 		"page":  "2",
	// 		"limit": "1",
	// 	}
	// 	resp, err := retrieveProductOptions("1", customFilter)
	// 	assert.Nil(t, err)
	// 	assertStatusCode(t, resp, http.StatusOK)

	// 	expected := models.ListResponse{
	// 		Limit: 1,
	// 		Page:  2,
	// 	}
	// 	var actual models.ListResponse
	// 	unmarshalBody(t, resp, &actual)
	// 	compareListResponses(t, expected, actual)
	// })
}

func TestProductOptionCreation(t *testing.T) {
	// t.Parallel()

	t.Run("normal usage", func(*testing.T) {
		testOptionName := "example_option_to_create"
		testSKU := "test-option-creation-sku"

		// create product to attach option to
		testProduct := models.ProductCreationInput{SKU: testSKU}
		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var createdProductRoot models.ProductRoot
		unmarshalBody(t, resp, &createdProductRoot)

		// create option
		testOption := models.ProductOptionCreationInput{
			Name:   testOptionName,
			Values: []string{"one", "two", "three"},
		}
		newOptionJSON := createJSONBody(t, testOption)
		resp, err = createProductOptionForProduct(strconv.Itoa(int(createdProductRoot.ID)), newOptionJSON)

		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		expected := models.ProductOption{
			Name:   testOptionName,
			Values: []models.ProductOptionValue{{Value: "one"}, {Value: "two"}, {Value: "three"}},
		}
		var actual models.ProductOption
		unmarshalBody(t, resp, &actual)
		compareProductOptions(t, expected, actual)
	})

	t.Run("with already existent name", func(*testing.T) {
		testOptionName := "already-existent-option"
		testSKU := "test-duplicate-option-sku"

		// create product to attach option to
		testProduct := models.ProductCreationInput{SKU: testSKU}
		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var createdProductRoot models.ProductRoot
		unmarshalBody(t, resp, &createdProductRoot)

		// create option
		testOption := models.ProductOptionCreationInput{
			Name:   testOptionName,
			Values: []string{"one", "two", "three"},
		}
		newOptionJSON := createJSONBody(t, testOption)
		resp, err = createProductOptionForProduct(strconv.Itoa(int(createdProductRoot.ID)), newOptionJSON)

		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		expected := models.ProductOption{
			Name:   testOptionName,
			Values: []models.ProductOptionValue{{Value: "one"}, {Value: "two"}, {Value: "three"}},
		}
		var actual models.ProductOption
		unmarshalBody(t, resp, &actual)
		compareProductOptions(t, expected, actual)

		// create option again
		resp, err = createProductOptionForProduct(strconv.Itoa(int(createdProductRoot.ID)), newOptionJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusBadRequest)
	})

	t.Run("with invalid input", func(*testing.T) {
		resp, err := createProductOptionForProduct(existentID, exampleGarbageInput)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusBadRequest)

		expected := models.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "Invalid input provided in request body",
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("for nonexistent product", func(*testing.T) {
		testOptionName := "already-existent-product"

		testOption := models.ProductOptionCreationInput{
			Name:   testOptionName,
			Values: []string{"one", "two", "three"},
		}
		newOptionJSON := createJSONBody(t, testOption)
		resp, err := createProductOptionForProduct(nonexistentID, newOptionJSON)

		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		expected := models.ErrorResponse{
			Status:  http.StatusNotFound,
			Message: fmt.Sprintf("The product root you were looking for (id '%s') does not exist", nonexistentID),
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})
}

func TestProductOptionDeletion(t *testing.T) {
	// t.Parallel()

	t.Run("normal usage", func(*testing.T) {
		testOptionName := "example_option_to_delete"
		testSKU := "test-option-deletion-sku"

		// create product to attach option to
		testProduct := models.ProductCreationInput{SKU: testSKU}
		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var createdProductRoot models.ProductRoot
		unmarshalBody(t, resp, &createdProductRoot)

		// create option
		testOption := models.ProductOptionCreationInput{
			Name:   testOptionName,
			Values: []string{"one", "two", "three"},
		}
		newOptionJSON := createJSONBody(t, testOption)
		resp, err = createProductOptionForProduct(strconv.Itoa(int(createdProductRoot.ID)), newOptionJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var created models.ProductOption
		unmarshalBody(t, resp, &created)

		// clean up after yourself
		resp, err = deleteProductOption(strconv.Itoa(int(created.ID)))

		var actual models.ProductOption
		assert.False(t, actual.ArchivedOn.Valid)
		unmarshalBody(t, resp, &actual)
		assert.True(t, actual.ArchivedOn.Valid)
	})

	t.Run("for nonexistent product option", func(*testing.T) {
		resp, err := deleteProductOption(nonexistentID)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		expected := models.ErrorResponse{
			Status:  http.StatusNotFound,
			Message: fmt.Sprintf("The product option you were looking for (id '%s') does not exist", nonexistentID),
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})
}

func TestProductOptionUpdate(t *testing.T) {
	// t.Parallel()

	t.Run("normal usage", func(*testing.T) {
		testSKU := "testing_product_options"
		testOptionName := "example_option_to_update"

		// create product to attach option to
		testProduct := models.ProductCreationInput{SKU: testSKU}
		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var createdProductRoot models.ProductRoot
		unmarshalBody(t, resp, &createdProductRoot)

		// create option
		testOption := models.ProductOptionCreationInput{
			Name:   testOptionName,
			Values: []string{"one", "two", "three"},
		}
		newOptionJSON := createJSONBody(t, testOption)
		resp, err = createProductOptionForProduct(strconv.Itoa(int(createdProductRoot.ID)), newOptionJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusCreated)

		var createdOption models.ProductOption
		unmarshalBody(t, resp, &createdOption)

		// update product option
		optionUpdate := models.ProductOption{Name: "not_the_same"} // `{"name": "not_the_same"}`
		optionUpdateJSON := createJSONBody(t, optionUpdate)
		resp, err = updateProductOption(strconv.Itoa(int(createdOption.ID)), optionUpdateJSON)

		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusOK)

		expected := models.ProductOption{
			Name:   "not_the_same",
			Values: []models.ProductOptionValue{{Value: "one"}, {Value: "two"}, {Value: "three"}},
		}

		var actual models.ProductOption
		assert.False(t, actual.UpdatedOn.Valid)
		unmarshalBody(t, resp, &actual)
		assert.True(t, actual.UpdatedOn.Valid)
		compareProductOptions(t, expected, actual)

		// clean up after yourself
		deleteProductOption(strconv.Itoa(int(createdOption.ID)))
	})

	t.Run("with invalid input", func(*testing.T) {
		resp, err := updateProductOption(existentID, exampleGarbageInput)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusBadRequest)

		expected := models.ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: "Invalid input provided in request body",
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("for nonexistent product option", func(*testing.T) {
		testOption := models.ProductOption{Name: "arbitrary"}
		testOptionJSON := createJSONBody(t, testOption)

		resp, err := updateProductOption(nonexistentID, testOptionJSON)
		assert.Nil(t, err)
		assertStatusCode(t, resp, http.StatusNotFound)

		expected := models.ErrorResponse{
			Status:  http.StatusNotFound,
			Message: fmt.Sprintf("The product option you were looking for (id '%s') does not exist", nonexistentID),
		}
		var actual models.ErrorResponse
		unmarshalBody(t, resp, &actual)
		assert.Equal(t, expected, actual)
	})
}

func TestProductOptionValueCreation(t *testing.T) {
	// t.Paralle()

	t.Run("normal usage", func(*testing.T) {
		testOptionName := "example_option_value_to_create"
		testSKU := "test-option-value-creation-sku"

		// create product to attach option to
		testProduct := models.ProductCreationInput{SKU: testSKU}
		newProductJSON := createJSONBody(t, testProduct)
		resp, err := createProduct(newProductJSON)
		require.Nil(t, err)

		var createdProductRoot models.ProductRoot
		unmarshalBody(t, resp, &createdProductRoot)

		// create option
		testOption := models.ProductOptionCreationInput{
			Name:   testOptionName,
			Values: []string{"one", "two", "three"},
		}
		newOptionJSON := createJSONBody(t, testOption)
		resp, err = createProductOptionForProduct(strconv.Itoa(int(createdProductRoot.ID)), newOptionJSON)
		require.Nil(t, err)

		expected := models.ProductOption{
			Name:   testOptionName,
			Values: []models.ProductOptionValue{{Value: "one"}, {Value: "two"}, {Value: "three"}},
		}
		var actual models.ProductOption
		unmarshalBody(t, resp, &actual)
		compareProductOptions(t, expected, actual)

		// newOptionValueJSON := createProductOptionValueBody(testValue)
		// resp, err := createProductOptionValueForOption(existentID, newOptionValueJSON)
		// assert.Nil(t, err)
		// assertStatusCode(t, resp, http.StatusCreated)
	})

	t.Run("with invalid input", func(*testing.T) {

	})

	t.Run("for nonexistent option", func(*testing.T) {

	})

	t.Run("with duplicate value", func(*testing.T) {

	})
}

func TestProductOptionValueUpdate(t *testing.T) {
	// t.Paralle()

	t.Run("normal usage", func(*testing.T) {
		// newOptionJSON := createProductOptionCreationBody(testOptionName)
		// createdProductRootIDString := strconv.Itoa(int(createdProductRootID))
		// resp, err := createProductOptionForProduct(createdProductRootIDString, newOptionJSON)
		// assert.Nil(t, err)
		// assertStatusCode(t, resp, http.StatusCreated)
	})

	t.Run("with invalid input", func(*testing.T) {

	})

	t.Run("for nonexistent option value", func(*testing.T) {

	})

	t.Run("with duplicate value", func(*testing.T) {
		// 	// Say you have a product option called `color`, and it has three values (`red`, `green`, and `blue`).
		// 	// Let's say you try to change `red` to `blue` for whatever reason. That will fail at the database level,
		// 	// because the schema ensures a unique combination of value and option ID and archived date.

	})
}

func TestProductOptionValueDeletion(t *testing.T) {
	// t.Paralle()

	t.Run("normal usage", func(*testing.T) {
		// newOptionValueJSON := createProductOptionValueBody(testValue)
		// resp, err := createProductOptionValueForOption(existentID, newOptionValueJSON)
		// assert.Nil(t, err)
		// assertStatusCode(t, resp, http.StatusCreated)
		// body := turnResponseBodyIntoString(t, resp)
		// createdOptionValueID = retrieveIDFromResponseBody(t, body)

		// resp, err := deleteProductOptionValueForOption(strconv.Itoa(int(createdOptionValueID)))
		// assert.Nil(t, err)
		// assertStatusCode(t, resp, http.StatusOK)
	})

	t.Run("for nonexistent option value", func(*testing.T) {
		// 	resp, err := deleteProductOptionValueForOption(nonexistentID)
		// 	assert.Nil(t, err)
		// 	assertStatusCode(t, resp, http.StatusNotFound)
	})
}
