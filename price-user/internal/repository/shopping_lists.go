package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceuser/internal/models"
)

var (
	ErrNotFound     = errors.New("shopping list not found")
	ErrLimitReached = errors.New("too many shopping lists")
)

type ShoppingListsRepo struct {
	pool *pgxpool.Pool
}

func NewShoppingListsRepo(pool *pgxpool.Pool) *ShoppingListsRepo {
	return &ShoppingListsRepo{pool: pool}
}

const shoppingListColumns = `id, name, jsonb_array_length(items), items, created_at, updated_at`

func scanShoppingList(row pgx.Row) (models.ShoppingList, error) {
	var l models.ShoppingList
	err := row.Scan(&l.ID, &l.Name, &l.ItemCount, &l.Items, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

// List returns the person's lists, most recently saved first.
func (r *ShoppingListsRepo) List(ctx context.Context, userID string) ([]models.ShoppingList, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shoppingListColumns+` FROM shopping_lists WHERE user_id = $1 ORDER BY updated_at DESC, name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lists := []models.ShoppingList{}
	for rows.Next() {
		l, err := scanShoppingList(rows)
		if err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}
	return lists, rows.Err()
}

// Save stores a list under name. A list the person already has under that name
// (ignoring case) is replaced; otherwise a new one is created, unless they
// already have maxLists. created reports which of the two happened.
func (r *ShoppingListsRepo) Save(ctx context.Context, userID, name string, items json.RawMessage, maxLists int) (list models.ShoppingList, created bool, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return list, false, err
	}
	defer tx.Rollback(ctx)

	var existingID string
	err = tx.QueryRow(ctx,
		`SELECT id FROM shopping_lists WHERE user_id = $1 AND lower(name) = lower($2) FOR UPDATE`,
		userID, name,
	).Scan(&existingID)

	switch {
	case err == nil:
		list, err = scanShoppingList(tx.QueryRow(ctx,
			`UPDATE shopping_lists SET name = $2, items = $3, updated_at = now() WHERE id = $1 RETURNING `+shoppingListColumns,
			existingID, name, items,
		))
	case errors.Is(err, pgx.ErrNoRows):
		var count int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM shopping_lists WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return list, false, err
		}
		if count >= maxLists {
			return list, false, ErrLimitReached
		}
		created = true
		list, err = scanShoppingList(tx.QueryRow(ctx,
			`INSERT INTO shopping_lists (user_id, name, items) VALUES ($1, $2, $3) RETURNING `+shoppingListColumns,
			userID, name, items,
		))
	}
	if err != nil {
		return list, false, err
	}

	return list, created, tx.Commit(ctx)
}

// Delete removes one of the person's lists. Someone else's list counts as not found.
func (r *ShoppingListsRepo) Delete(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM shopping_lists WHERE id::text = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
