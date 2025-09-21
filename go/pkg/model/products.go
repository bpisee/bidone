package model

import (
	"errors"
	"strings"
	"time"
)

// Product represents an item in inventory.
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	PriceCents  int64     `json:"price_cents"`
	Quantity    int64     `json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var (
	errInvalidName     = errors.New("name must not be empty")
	errInvalidPrice    = errors.New("price_cents must be >= 0")
	errInvalidQuantity = errors.New("quantity must be >= 0")
)

func (p *Product) ValidateForCreate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errInvalidName
	}
	if p.PriceCents < 0 {
		return errInvalidPrice
	}
	if p.Quantity < 0 {
		return errInvalidQuantity
	}
	return nil
}

func (p *Product) ValidateForUpdate() error {
	// Same as create for this simple model; could be relaxed if partial updates allowed
	return p.ValidateForCreate()
}
