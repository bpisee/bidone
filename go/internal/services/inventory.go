package services

import (
	"context"
	"strings"

	"github.com/lee/BidOne/internal/store"
	"github.com/lee/BidOne/pkg/model"
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

// inventoryService is the concrete implementation of InventoryService
// that uses a MemoryStore for data persistence.
type inventoryService struct {
	store *store.MemoryStore
}

// NewInventoryService creates a new InventoryService instance with the provided store.
// The store parameter must not be nil.
func NewInventoryService(store *store.MemoryStore) InventoryService {
	return &inventoryService{
		store: store,
	}
}

// ListProducts implements the InventoryService interface.
// It retrieves all products from the store, applies name filtering if specified,
// and returns a paginated subset of the results.
func (s *inventoryService) ListProducts(ctx context.Context, limit, offset int, name string) ([]model.Product, error) {
	// Validate pagination parameters
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Maximum limit to prevent excessive memory usage
	}
	if offset < 0 {
		offset = 0
	}

	// Retrieve all products from the store
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	// Apply name filtering if specified
	filtered := make([]model.Product, 0, len(items))
	nameFilter := strings.ToLower(strings.TrimSpace(name))

	for _, product := range items {
		// If no name filter is specified, include all products
		if nameFilter == "" {
			filtered = append(filtered, product)
			continue
		}

		// Case-insensitive substring matching on product name
		if strings.Contains(strings.ToLower(product.Name), nameFilter) {
			filtered = append(filtered, product)
		}
	}

	// Apply pagination
	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	// Return the paginated slice
	return filtered[start:end], nil
}

// GetProduct implements the InventoryService interface.
// It retrieves a single product by ID from the store.
func (s *inventoryService) GetProduct(ctx context.Context, id string) (model.Product, error) {
	return s.store.Get(ctx, id)
}

// CreateProduct implements the InventoryService interface.
// It validates the product and creates it in the store.
func (s *inventoryService) CreateProduct(ctx context.Context, p model.Product) (model.Product, error) {
	return s.store.Create(ctx, p)
}

// UpdateProduct implements the InventoryService interface.
// It validates the product and updates it in the store.
func (s *inventoryService) UpdateProduct(ctx context.Context, id string, p model.Product) (model.Product, error) {
	return s.store.Update(ctx, id, p)
}

// DeleteProduct implements the InventoryService interface.
// It removes the product from the store.
func (s *inventoryService) DeleteProduct(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
