package services

import (
	"context"
	"testing"

	"github.com/lee/BidOne/apps/inventorysvc/store"
	"github.com/lee/BidOne/shared/types/model"
)

func TestNewInventoryService(t *testing.T) {
	store := store.NewMemoryStore()
	service := NewInventoryService(store)

	if service == nil {
		t.Fatal("NewInventoryService returned nil")
	}
}

func TestInventoryService_CreateProduct(t *testing.T) {
	store := store.NewMemoryStore()
	service := NewInventoryService(store)
	ctx := context.Background()

	t.Run("ValidProduct", func(t *testing.T) {
		product := model.Product{
			Name:        "Test Widget",
			Description: "A test widget for testing",
			PriceCents:  1999,
			Quantity:    10,
		}

		created, err := service.CreateProduct(ctx, product)
		if err != nil {
			t.Fatalf("CreateProduct failed: %v", err)
		}

		// Verify the product was created with proper fields
		if created.ID == "" {
			t.Error("Created product should have an ID")
		}
		if created.Name != product.Name {
			t.Errorf("Expected name %q, got %q", product.Name, created.Name)
		}
		if created.PriceCents != product.PriceCents {
			t.Errorf("Expected price %d, got %d", product.PriceCents, created.PriceCents)
		}
		if created.Quantity != product.Quantity {
			t.Errorf("Expected quantity %d, got %d", product.Quantity, created.Quantity)
		}
		if created.CreatedAt.IsZero() {
			t.Error("Created product should have CreatedAt timestamp")
		}
		if created.UpdatedAt.IsZero() {
			t.Error("Created product should have UpdatedAt timestamp")
		}
	})

	t.Run("InvalidProduct", func(t *testing.T) {
		// Test with empty name
		product := model.Product{
			Name:       "", // Invalid: empty name
			PriceCents: 1999,
			Quantity:   10,
		}

		_, err := service.CreateProduct(ctx, product)
		if err == nil {
			t.Error("CreateProduct should fail with empty name")
		}
	})

	t.Run("NegativePrice", func(t *testing.T) {
		product := model.Product{
			Name:       "Test Widget",
			PriceCents: -100, // Invalid: negative price
			Quantity:   10,
		}

		_, err := service.CreateProduct(ctx, product)
		if err == nil {
			t.Error("CreateProduct should fail with negative price")
		}
	})

	t.Run("NegativeQuantity", func(t *testing.T) {
		product := model.Product{
			Name:       "Test Widget",
			PriceCents: 1999,
			Quantity:   -5, // Invalid: negative quantity
		}

		_, err := service.CreateProduct(ctx, product)
		if err == nil {
			t.Error("CreateProduct should fail with negative quantity")
		}
	})
}

func TestInventoryService_GetProduct(t *testing.T) {
	store := store.NewMemoryStore()
	service := NewInventoryService(store)
	ctx := context.Background()

	// Create a product first
	product := model.Product{
		Name:       "Test Widget",
		PriceCents: 1999,
		Quantity:   10,
	}

	created, err := service.CreateProduct(ctx, product)
	if err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	t.Run("ExistingProduct", func(t *testing.T) {
		retrieved, err := service.GetProduct(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetProduct failed: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected ID %q, got %q", created.ID, retrieved.ID)
		}
		if retrieved.Name != created.Name {
			t.Errorf("Expected name %q, got %q", created.Name, retrieved.Name)
		}
	})

	t.Run("NonExistentProduct", func(t *testing.T) {
		_, err := service.GetProduct(ctx, "non-existent-id")
		if err == nil {
			t.Error("GetProduct should fail for non-existent product")
		}
	})
}

func TestInventoryService_UpdateProduct(t *testing.T) {
	store := store.NewMemoryStore()
	service := NewInventoryService(store)
	ctx := context.Background()

	// Create a product first
	original := model.Product{
		Name:       "Original Widget",
		PriceCents: 1999,
		Quantity:   10,
	}

	created, err := service.CreateProduct(ctx, original)
	if err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	t.Run("ValidUpdate", func(t *testing.T) {
		updated := model.Product{
			Name:        "Updated Widget",
			Description: "Updated description",
			PriceCents:  2999,
			Quantity:    15,
		}

		result, err := service.UpdateProduct(ctx, created.ID, updated)
		if err != nil {
			t.Fatalf("UpdateProduct failed: %v", err)
		}

		// Verify the update
		if result.ID != created.ID {
			t.Errorf("ID should remain the same: expected %q, got %q", created.ID, result.ID)
		}
		if result.Name != updated.Name {
			t.Errorf("Expected name %q, got %q", updated.Name, result.Name)
		}
		if result.PriceCents != updated.PriceCents {
			t.Errorf("Expected price %d, got %d", updated.PriceCents, result.PriceCents)
		}
		if result.CreatedAt != created.CreatedAt {
			t.Error("CreatedAt should remain unchanged")
		}
		if result.UpdatedAt.Before(created.UpdatedAt) {
			t.Error("UpdatedAt should not be older than original")
		}
		// Note: UpdatedAt might be equal if the update happens very quickly,
		// but it should never be older than the original
	})

	t.Run("NonExistentProduct", func(t *testing.T) {
		updated := model.Product{
			Name:       "Updated Widget",
			PriceCents: 2999,
			Quantity:   15,
		}

		_, err := service.UpdateProduct(ctx, "non-existent-id", updated)
		if err == nil {
			t.Error("UpdateProduct should fail for non-existent product")
		}
	})

	t.Run("InvalidUpdate", func(t *testing.T) {
		invalid := model.Product{
			Name:       "", // Invalid: empty name
			PriceCents: 2999,
			Quantity:   15,
		}

		_, err := service.UpdateProduct(ctx, created.ID, invalid)
		if err == nil {
			t.Error("UpdateProduct should fail with invalid product data")
		}
	})
}

func TestInventoryService_DeleteProduct(t *testing.T) {
	store := store.NewMemoryStore()
	service := NewInventoryService(store)
	ctx := context.Background()

	// Create a product first
	product := model.Product{
		Name:       "Test Widget",
		PriceCents: 1999,
		Quantity:   10,
	}

	created, err := service.CreateProduct(ctx, product)
	if err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	t.Run("ExistingProduct", func(t *testing.T) {
		err := service.DeleteProduct(ctx, created.ID)
		if err != nil {
			t.Fatalf("DeleteProduct failed: %v", err)
		}

		// Verify the product is gone
		_, err = service.GetProduct(ctx, created.ID)
		if err == nil {
			t.Error("Product should be deleted and not retrievable")
		}
	})

	t.Run("NonExistentProduct", func(t *testing.T) {
		err := service.DeleteProduct(ctx, "non-existent-id")
		if err == nil {
			t.Error("DeleteProduct should fail for non-existent product")
		}
	})
}

func TestInventoryService_ListProducts(t *testing.T) {
	store := store.NewMemoryStore()
	service := NewInventoryService(store)
	ctx := context.Background()

	// Create test products
	testProducts := []model.Product{
		{Name: "Apple Widget", PriceCents: 1000, Quantity: 5},
		{Name: "Banana Gadget", PriceCents: 2000, Quantity: 10},
		{Name: "Cherry Tool", PriceCents: 3000, Quantity: 15},
		{Name: "Date Device", PriceCents: 4000, Quantity: 20},
		{Name: "Elderberry Equipment", PriceCents: 5000, Quantity: 25},
	}

	var createdProducts []model.Product
	for _, p := range testProducts {
		created, err := service.CreateProduct(ctx, p)
		if err != nil {
			t.Fatalf("Failed to create test product %q: %v", p.Name, err)
		}
		createdProducts = append(createdProducts, created)
	}

	t.Run("ListAllProducts", func(t *testing.T) {
		products, err := service.ListProducts(ctx, 100, 0, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		if len(products) != len(testProducts) {
			t.Errorf("Expected %d products, got %d", len(testProducts), len(products))
		}
	})

	t.Run("PaginationLimit", func(t *testing.T) {
		products, err := service.ListProducts(ctx, 2, 0, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		if len(products) != 2 {
			t.Errorf("Expected 2 products with limit=2, got %d", len(products))
		}
	})

	t.Run("PaginationOffset", func(t *testing.T) {
		// Get first page
		firstPage, err := service.ListProducts(ctx, 2, 0, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		// Get second page
		secondPage, err := service.ListProducts(ctx, 2, 2, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		// Verify page sizes
		if len(firstPage) != 2 {
			t.Errorf("First page should have 2 products, got %d", len(firstPage))
		}
		if len(secondPage) != 2 {
			t.Errorf("Second page should have 2 products, got %d", len(secondPage))
		}

		// Get all products to verify total count
		allProducts, err := service.ListProducts(ctx, 100, 0, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		// Verify that first page + second page + remaining = total
		totalFromPages := len(firstPage) + len(secondPage)
		if totalFromPages > len(allProducts) {
			t.Errorf("Pages contain more products (%d) than total available (%d)", totalFromPages, len(allProducts))
		}
	})

	t.Run("NameFiltering", func(t *testing.T) {
		// Filter by "berry" should match "Elderberry Equipment"
		products, err := service.ListProducts(ctx, 100, 0, "berry")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		if len(products) != 1 {
			t.Errorf("Expected 1 product matching 'berry', got %d", len(products))
		}

		if len(products) > 0 && products[0].Name != "Elderberry Equipment" {
			t.Errorf("Expected 'Elderberry Equipment', got %q", products[0].Name)
		}
	})

	t.Run("CaseInsensitiveFiltering", func(t *testing.T) {
		// Filter by "APPLE" should match "Apple Widget" (case insensitive)
		products, err := service.ListProducts(ctx, 100, 0, "APPLE")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		if len(products) != 1 {
			t.Errorf("Expected 1 product matching 'APPLE', got %d", len(products))
		}

		if len(products) > 0 && products[0].Name != "Apple Widget" {
			t.Errorf("Expected 'Apple Widget', got %q", products[0].Name)
		}
	})

	t.Run("NoMatchesFilter", func(t *testing.T) {
		products, err := service.ListProducts(ctx, 100, 0, "nonexistent")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		if len(products) != 0 {
			t.Errorf("Expected 0 products matching 'nonexistent', got %d", len(products))
		}
	})

	t.Run("DefaultLimitValidation", func(t *testing.T) {
		// Test with invalid limit (0 or negative)
		products, err := service.ListProducts(ctx, 0, 0, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		// Should use default limit and return all products (since we have < 100)
		if len(products) != len(testProducts) {
			t.Errorf("Expected %d products with default limit, got %d", len(testProducts), len(products))
		}
	})

	t.Run("MaximumLimitValidation", func(t *testing.T) {
		// Test with excessive limit (> 1000)
		products, err := service.ListProducts(ctx, 2000, 0, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		// Should still return all products (limit should be capped at 1000)
		if len(products) != len(testProducts) {
			t.Errorf("Expected %d products with capped limit, got %d", len(testProducts), len(products))
		}
	})

	t.Run("NegativeOffsetValidation", func(t *testing.T) {
		products, err := service.ListProducts(ctx, 100, -10, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		// Should treat negative offset as 0
		if len(products) != len(testProducts) {
			t.Errorf("Expected %d products with corrected offset, got %d", len(testProducts), len(products))
		}
	})

	t.Run("OffsetBeyondResults", func(t *testing.T) {
		products, err := service.ListProducts(ctx, 100, 1000, "")
		if err != nil {
			t.Fatalf("ListProducts failed: %v", err)
		}

		// Should return empty slice when offset is beyond available results
		if len(products) != 0 {
			t.Errorf("Expected 0 products with offset beyond results, got %d", len(products))
		}
	})
}
