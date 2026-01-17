package repo

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/internal/inventory/domain"
	"github.com/k-code-yt/go-api-practice/kafka-cdc/pkg/db/postgres"
)

type InventoryRepo struct {
	repo      *sqlx.DB
	tableName string
}

func NewInventoryRepo(db *sqlx.DB) *InventoryRepo {
	return &InventoryRepo{
		repo:      db,
		tableName: DBTableName_Inventory,
	}
}

func (r *InventoryRepo) GetRepo() *sqlx.DB {
	return r.repo
}

func (r *InventoryRepo) Insert(ctx context.Context, tx *sqlx.Tx, inv *domain.Inventory) (int, error) {
	query := fmt.Sprintf(`INSERT INTO %s ("product_name", "status", "quantity", "last_updated", "order_number", "payment_id") VALUES($1, $2, $3, $4, $5, $6) RETURNING id`, r.tableName)
	rows, err := tx.QueryxContext(ctx, query, inv.ProductName, inv.Status, inv.Quantity, inv.LastUpdated, inv.OrderNumber, inv.PaymentId)
	if err != nil {
		return postgres.NonExistingIntKey, err
	}
	defer rows.Close()
	var id int
	for rows.Next() {
		ids, err := rows.SliceScan()
		if err != nil {
			fmt.Printf("unable to get last insetID = %v\n", err)
			return postgres.NonExistingIntKey, err
		}
		id = int(ids[0].(int64))
	}
	fmt.Printf("Inserted inventoryID = %d\n", inv.ID)
	return id, nil
}
