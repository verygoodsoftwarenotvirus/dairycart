package main

import (
	"database/sql"
	"time"
)

const (
	productProgenitorExistenceQuery = `SELECT EXISTS(SELECT 1 FROM product_progenitors WHERE id = $1 AND archived_on IS NULL)`
	productProgenitorRetrievalQuery = `SELECT * FROM product_progenitors WHERE id = $1`
)

// ProductProgenitor is the parent product for every product
type ProductProgenitor struct {
	// Basic Info
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Pricing Fields
	Taxable bool    `json:"taxable"`
	Price   float32 `json:"price"`
	Cost    float32 `json:"cost"`

	// Product Dimensions
	ProductWeight float32 `json:"product_weight"`
	ProductHeight float32 `json:"product_height"`
	ProductWidth  float32 `json:"product_width"`
	ProductLength float32 `json:"product_length"`

	// Package dimensions
	PackageWeight float32 `json:"package_weight"`
	PackageHeight float32 `json:"package_height"`
	PackageWidth  float32 `json:"package_width"`
	PackageLength float32 `json:"package_length"`

	// // Housekeeping
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  NullTime  `json:"updated_on,omitempty"`
	ArchivedOn NullTime  `json:"archived_on,omitempty"`
}

// generateScanArgs generates an array of pointers to struct fields for sql.Scan to populate
func (g *ProductProgenitor) generateScanArgs() []interface{} {
	return []interface{}{
		&g.ID,
		&g.Name,
		&g.Description,
		&g.Taxable,
		&g.Price,
		&g.Cost,
		&g.ProductWeight,
		&g.ProductHeight,
		&g.ProductWidth,
		&g.ProductLength,
		&g.PackageWeight,
		&g.PackageHeight,
		&g.PackageWidth,
		&g.PackageLength,
		&g.CreatedOn,
		&g.UpdatedOn,
		&g.ArchivedOn,
	}
}

func newProductProgenitorFromProductCreationInput(in *ProductCreationInput) *ProductProgenitor {
	return &ProductProgenitor{
		Name:          in.Name,
		Description:   in.Description,
		Taxable:       in.Taxable,
		Price:         in.Price,
		Cost:          in.Cost,
		ProductWeight: in.ProductWeight,
		ProductHeight: in.ProductHeight,
		ProductWidth:  in.ProductWidth,
		ProductLength: in.ProductLength,
		PackageWeight: in.PackageWeight,
		PackageHeight: in.PackageHeight,
		PackageWidth:  in.PackageWidth,
		PackageLength: in.PackageLength,
	}
}

func createProductProgenitorInDB(tx *sql.Tx, g *ProductProgenitor) (int64, error) {
	var newProgenitorID int64
	// using QueryRow instead of Exec because we want it to return the newly created row's ID
	// Exec normally returns a sql.Result, which has a LastInsertedID() method, but when I tested
	// this locally, it never worked. ¯\_(ツ)_/¯
	query, queryArgs := buildProgenitorCreationQuery(g)
	err := tx.QueryRow(query, queryArgs...).Scan(&newProgenitorID)

	return newProgenitorID, err
}

// retrieveProductProgenitorFromDB retrieves a product progenitor with a given ID from the database
func retrieveProductProgenitorFromDB(db *sql.DB, id int64) (*ProductProgenitor, error) {
	progenitor := &ProductProgenitor{}
	scanArgs := progenitor.generateScanArgs()

	err := db.QueryRow(productProgenitorRetrievalQuery, id).Scan(scanArgs...)

	return progenitor, err
}
