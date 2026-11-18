package models

import (
	"encoding/json"
	"time"
)

// ShoppingList is a list a person saved from the app. Items is kept exactly as
// the app sent it (product snapshot + quantity), so the app can put it back in
// the cart as it was.
type ShoppingList struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	ItemCount int             `json:"item_count"`
	Items     json.RawMessage `json:"items"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
