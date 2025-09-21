package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lee/BidOne/internal/services"
	"github.com/lee/BidOne/internal/store"
	"github.com/lee/BidOne/pkg/model"
)

func TestCreateAndListProducts(t *testing.T) {
	// Create the data store and service layer
	st := store.NewMemoryStore()
	inventoryService := services.NewInventoryService(st)
	s := NewServer(inventoryService)
	e := s.Echo()

	// Test creating a product (using the request format, not the full model)
	createReq := map[string]interface{}{
		"name":        "Widget",
		"price_cents": 1999,
		"quantity":    5,
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	// Test listing products
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/products?limit=10&offset=0", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec2.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if int(resp["total"].(float64)) != 1 {
		t.Fatalf("expected total 1, got %v", resp["total"])
	}
}

func TestServiceLayerIntegration(t *testing.T) {
	// Create the data store and service layer
	st := store.NewMemoryStore()
	inventoryService := services.NewInventoryService(st)
	s := NewServer(inventoryService)
	e := s.Echo()

	// Test creating multiple products (using request format)
	testProductRequests := []map[string]interface{}{
		{"name": "Apple iPhone", "price_cents": 99999, "quantity": 10},
		{"name": "Samsung Galaxy", "price_cents": 89999, "quantity": 15},
		{"name": "Google Pixel", "price_cents": 79999, "quantity": 8},
	}

	var createdIDs []string
	for _, p := range testProductRequests {
		body, _ := json.Marshal(p)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", rec.Code)
		}

		var created model.Product
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		createdIDs = append(createdIDs, created.ID)
	}

	// Test listing all products
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var listResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if int(listResp["total"].(float64)) != 3 {
		t.Fatalf("expected total 3, got %v", listResp["total"])
	}

	// Test filtering by name
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products?name=apple", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var filterResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &filterResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if int(filterResp["total"].(float64)) != 1 {
		t.Fatalf("expected filtered total 1, got %v", filterResp["total"])
	}

	// Test pagination
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products?limit=2&offset=0", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var pageResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pageResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	productsArray := pageResp["products"].([]interface{})
	if len(productsArray) != 2 {
		t.Fatalf("expected 2 products in page, got %d", len(productsArray))
	}

	// Test getting a specific product
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products/"+createdIDs[0], nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Test updating a product (using request format)
	updateReq := map[string]interface{}{
		"name":        "Apple iPhone Pro",
		"price_cents": 109999,
		"quantity":    12,
	}
	body, _ := json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/products/"+createdIDs[0], bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Test deleting a product
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+createdIDs[0], nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	// Verify deletion - should have 2 products left
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var finalResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &finalResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if int(finalResp["total"].(float64)) != 2 {
		t.Fatalf("expected total 2 after deletion, got %v", finalResp["total"])
	}
}

func TestHealthCheck(t *testing.T) {
	// Create the data store and service layer
	st := store.NewMemoryStore()
	inventoryService := services.NewInventoryService(st)
	s := NewServer(inventoryService)
	e := s.Echo()

	// Test health check endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var healthResp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if healthResp["status"] != "healthy" {
		t.Fatalf("expected status 'healthy', got %v", healthResp["status"])
	}

	if healthResp["service"] != "inventory-service" {
		t.Fatalf("expected service 'inventory-service', got %v", healthResp["service"])
	}
}

func TestCreateProductValidation(t *testing.T) {
	// Create the data store and service layer
	st := store.NewMemoryStore()
	inventoryService := services.NewInventoryService(st)
	s := NewServer(inventoryService)
	e := s.Echo()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid product creation",
			requestBody: map[string]interface{}{
				"name":        "Valid Product",
				"price_cents": 2999,
				"quantity":    10,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing name",
			requestBody: map[string]interface{}{
				"price_cents": 2999,
				"quantity":    10,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name is required",
		},
		{
			name: "Empty name",
			requestBody: map[string]interface{}{
				"name":        "",
				"price_cents": 2999,
				"quantity":    10,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name is required",
		},
		{
			name: "Whitespace only name",
			requestBody: map[string]interface{}{
				"name":        "   ",
				"price_cents": 2999,
				"quantity":    10,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name must not be empty",
		},
		{
			name: "Negative price",
			requestBody: map[string]interface{}{
				"name":        "Valid Product",
				"price_cents": -100,
				"quantity":    10,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "price_cents is required and must be >= 0",
		},
		{
			name: "Negative quantity",
			requestBody: map[string]interface{}{
				"name":        "Valid Product",
				"price_cents": 2999,
				"quantity":    -5,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "quantity is required and must be >= 0",
		},
		{
			name: "Multiple validation errors",
			requestBody: map[string]interface{}{
				"name":        "",
				"price_cents": -100,
				"quantity":    -5,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation failed",
		},
		{
			name: "Zero values (should be valid)",
			requestBody: map[string]interface{}{
				"name":        "Free Product",
				"price_cents": 0,
				"quantity":    0,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "With description",
			requestBody: map[string]interface{}{
				"name":        "Product with Description",
				"description": "This is a test product",
				"price_cents": 1999,
				"quantity":    5,
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Response: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			if tt.expectedError != "" {
				var errorResp map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &errorResp); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}
				if !strings.Contains(errorResp["error"], tt.expectedError) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.expectedError, errorResp["error"])
				}
			}

			// For successful creations, verify the response structure
			if tt.expectedStatus == http.StatusCreated {
				var product model.Product
				if err := json.Unmarshal(rec.Body.Bytes(), &product); err != nil {
					t.Fatalf("failed to unmarshal product response: %v", err)
				}

				// Verify that ID was auto-generated
				if product.ID == "" {
					t.Error("expected auto-generated ID, got empty string")
				}

				// Verify timestamps were set
				if product.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if product.UpdatedAt.IsZero() {
					t.Error("expected UpdatedAt to be set")
				}
			}
		})
	}
}
