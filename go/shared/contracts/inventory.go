package contracts

import (
	"context"

	"github.com/lee/BidOne/shared/types/model"
)


// InventoryService defines the interface for inventory management operations.
// It provides business logic for product management including filtering,
// pagination, and validation.
type InventoryService interface {
	// ListProducts retrieves a paginated list of products with optional name filtering.
	// The name parameter performs case-insensitive substring matching.
	// Returns the filtered and paginated slice of model.
	ListProducts(ctx context.Context, limit, offset int, name string) ([]model.Product, error)

	// GetProduct retrieves a single product by its ID.
	// Returns the product if found, or an error if not found.
	GetProduct(ctx context.Context, id string) (model.Product, error)

	// CreateProduct creates a new product in the inventory.
	// The product will be validated and assigned a new ID and timestamps.
	CreateProduct(ctx context.Context, p model.Product) (model.Product, error)

	// UpdateProduct updates an existing product by ID.
	// The product will be validated and the updated_at timestamp will be refreshed.
	UpdateProduct(ctx context.Context, id string, p model.Product) (model.Product, error)

	// DeleteProduct removes a product from the inventory by ID.
	// Returns an error if the product is not found.
	DeleteProduct(ctx context.Context, id string) error
}