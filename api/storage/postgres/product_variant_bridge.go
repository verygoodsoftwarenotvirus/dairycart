package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dairycart/dairycart/api/storage"
	"github.com/dairycart/dairycart/api/storage/models"
)

const productVariantBridgeExistenceQuery = `SELECT EXISTS(SELECT id FROM product_variant_bridge WHERE id = $1 and archived_on IS NULL);`

func (pg *postgres) ProductVariantBridgeExists(db storage.Querier, id uint64) (bool, error) {
	var exists string

	err := db.QueryRow(productVariantBridgeExistenceQuery, id).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return exists == "true", err
}

const productVariantBridgeSelectionQuery = `
    SELECT
        id,
        product_id,
        product_option_value_id,
        created_on,
        archived_on
    FROM
        product_variant_bridge
    WHERE
        archived_on is null
    AND
        id = $1
`

func (pg *postgres) GetProductVariantBridge(db storage.Querier, id uint64) (*models.ProductVariantBridge, error) {
	p := &models.ProductVariantBridge{}

	err := db.QueryRow(productVariantBridgeSelectionQuery, id).Scan(&p.ID, &p.ProductID, &p.ProductOptionValueID, &p.CreatedOn, &p.ArchivedOn)

	return p, err
}

const productvariantbridgeCreationQuery = `
    INSERT INTO product_variant_bridge
        (
            product_id, product_option_value_id
        )
    VALUES
        (
            $1, $2
        )
    RETURNING
        id, created_on;
`

func (pg *postgres) CreateProductVariantBridge(db storage.Querier, nu *models.ProductVariantBridge) (uint64, time.Time, error) {
	var (
		createdID uint64
		createdAt time.Time
	)

	err := db.QueryRow(productvariantbridgeCreationQuery, &nu.ProductID, &nu.ProductOptionValueID).Scan(&createdID, &createdAt)

	return createdID, createdAt, err
}
func buildMultiProductVariantBridgeCreationQuery(productID uint64, optionValueIDs []uint64) (query string, values []interface{}) {
	values = append(values, productID)
	var valueString string
	for ix, id := range optionValueIDs {
		if ix == 0 {
			valueString = fmt.Sprintf("($1, $%d)", ix+2)
			values = append(values, id)
		} else {
			valueString = fmt.Sprintf(`%s
                ($1, $%d)`, valueString, ix+2)
			values = append(values, id)
		}
		if len(optionValueIDs)-1 != ix {
			valueString = fmt.Sprintf("%s,", valueString)
		}
	}

	query = fmt.Sprintf(`
        INSERT INTO product_variant_bridge
            (
                product_id, product_option_value_id
            )
        VALUES
            (
                %s
            )
        RETURNING
            id, created_on;
    `, valueString)

	return query, values
}

func (pg *postgres) CreateMultipleProductVariantBridgesForProductID(db storage.Querier, productID uint64, optionValueIDs []uint64) error {
	query, args := buildMultiProductVariantBridgeCreationQuery(productID, optionValueIDs)
	_, err := db.Exec(query, args...)
	return err
}

const productVariantBridgeUpdateQuery = `
    UPDATE product_variant_bridge
    SET
        product_id = $1, 
        product_option_value_id = $2
    WHERE id = $2
    RETURNING updated_on;
`

func (pg *postgres) UpdateProductVariantBridge(db storage.Querier, updated *models.ProductVariantBridge) (time.Time, error) {
	var t time.Time
	err := db.QueryRow(productVariantBridgeUpdateQuery, &updated.ProductID, &updated.ProductOptionValueID, &updated.ID).Scan(&t)
	return t, err
}

const productVariantBridgeDeletionQuery = `
    UPDATE product_variant_bridge
    SET archived_on = NOW()
    WHERE id = $1
    RETURNING archived_on
`

func (pg *postgres) DeleteProductVariantBridge(db storage.Querier, id uint64) (t time.Time, err error) {
	err = db.QueryRow(productVariantBridgeDeletionQuery, id).Scan(&t)
	return t, err
}

const productVariantBridgeDeletionQueryByProductID = `
    UPDATE product_variant_bridge SET archived_on = NOW() WHERE product_id = $1 AND archived_on IS NULL RETURNING archived_on
`

func (pg *postgres) DeleteProductVariantBridgeByProductID(db storage.Querier, productID uint64) (t time.Time, err error) {
	err = db.QueryRow(productVariantBridgeDeletionQueryByProductID, productID).Scan(&t)
	return t, err
}