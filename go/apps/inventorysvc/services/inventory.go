package services

import (
	"context"
	"strings"

	"github.com/lee/BidOne/apps/inventorysvc/store"
	"github.com/lee/BidOne/shared/contracts"
	"github.com/lee/BidOne/shared/types/model"
)

// inventoryService is the concrete implementation of InventoryService
// that uses a MemoryStore for data persistence.
type inventoryService struct {
	store *store.MemoryStore
}

// NewInventoryService creates a new InventoryService instance with the provided store.
// The store parameter must not be nil.
func NewInventoryService(store *store.MemoryStore) contracts.InventoryService {
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
