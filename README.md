# BidOne - Inventory Service

A high-performance inventory management service built with Go and Echo framework, featuring comprehensive metrics collection using goroutines and channels.

## Quick Start

### Prerequisites
- Go 1.19 or later
- Git

### Installation & Setup

```bash
# Clone the repository
git clone https://github.com/bpisee/bidone.git
cd bidone/go

# Initialize Go workspace (if needed)
go work init

# Add modules to workspace
go work use ./apps/inventorysvc
go work use ./shared

# Install dependencies
cd apps/inventorysvc
go mod tidy
```

## Running the Inventory Service

### Build and Run

#### Navigate to the inventory service directory
```bash
cd go/apps/inventorysvc
```

#### Build the service
```bash
# Standard build
go build -o bin/inventorysvc.exe .

# Optimized build for production
go build -ldflags="-s -w" -o bin/inventorysvc-optimized.exe .
```

#### Run the service
```bash
# Run standard build
./bin/inventorysvc.exe

# Or run optimized build
./bin/inventorysvc-optimized.exe
```

The service will start on `http://localhost:8080` with the following endpoints:

### API Endpoints
- **Health Check**: `GET /health`
- **Products API**: `GET|POST|PUT|DELETE /api/v1/products`
- **Metrics API**: `GET /metrics`, `GET /metrics/summary`, `POST /metrics/reset`

### Example API Usage

#### Using PowerShell (Windows)
```powershell
# Health check
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET

# Create a product
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/products" -Method POST `
  -ContentType "application/json" `
  -Body '{"name":"Test Product","description":"A sample product","price_cents":2999,"quantity":10}'

# List products
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/products" -Method GET

# Get specific product
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/products/{product-id}" -Method GET

# Get metrics summary
Invoke-RestMethod -Uri "http://localhost:8080/metrics/summary" -Method GET
```

#### Using curl (Linux/macOS)
```bash
# Health check
curl http://localhost:8080/health

# Create a product
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Product","price_cents":2999,"quantity":10}'

# List products
curl http://localhost:8080/api/v1/products
```

## Testing

### Run All Tests

#### From the inventory service directory (`go/apps/inventorysvc/`)
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -v -bench=. ./...

# Run specific package tests
go test -v ./services/
go test -v ./store/
```

#### From the root go directory (`go/`)
```bash
# Run tests for shared packages
go test -v ./shared/pkg/metrics/
go test -v ./shared/pkg/ratelimiter/
```

## Key Features

### 🚀 **High Performance**
- Echo web framework for fast HTTP handling
- Goroutine-based metrics collection
- Non-blocking channel communication
- Thread-safe concurrent operations

### 📊 **Comprehensive Metrics**
- **HTTP Metrics**: Request/response tracking, latency, status codes
- **Business Metrics**: Product operations, inventory levels, validation errors
- **Real-time Collection**: Background goroutines with buffered channels
- **API Endpoints**: `/metrics`, `/metrics/summary`, `/metrics/reset`

### ✅ **Production Ready**
- Input validation with Echo's built-in validator
- Structured error handling and logging
- Graceful shutdown with proper cleanup
- Comprehensive test coverage

### 🏗️ **Clean Architecture**
- Service layer separation
- Interface-based design
- Dependency injection
- Modular package structure

## Project Structure

```
bidone/
├── go/                        # Go workspace root
│   ├── go.work               # Go workspace configuration
│   ├── apps/                 # Application services
│   │   ├── inventorysvc/     # Inventory service
│   │   │   ├── main.go       # Service entry point
│   │   │   ├── router.go     # HTTP routing
│   │   │   ├── httphandler.go # HTTP handlers
│   │   │   ├── services/     # Business logic layer
│   │   │   ├── store/        # Data access layer
│   │   │   └── bin/          # Built binaries
│   │   └── ordersvc/         # Order service (future)
│   └── shared/               # Shared packages
│       └── pkg/              # Reusable packages
│           ├── metrics/      # Metrics collection
│           └── ratelimiter/  # Rate limiting
└── answers/                  # Documentation and examples
    └── CODEREVIEW.md        # Code review examples
```

## Development

### Code Quality

#### From the inventory service directory (`go/apps/inventorysvc/`)
```bash
# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run

# Vet code
go vet ./...
```

### Building for Production

#### From the inventory service directory (`go/apps/inventorysvc/`)
```bash
# Build optimized binary
go build -ldflags="-s -w" -o bin/inventorysvc-optimized.exe .

# Build with version information
go build -ldflags="-s -w -X main.version=1.0.0" -o bin/inventorysvc.exe .
```

## Go Workspace Structure

This project uses Go workspaces to manage multiple modules:

- **`go.work`**: Workspace configuration file
- **`apps/`**: Individual service applications
- **`shared/`**: Shared packages and utilities

### Working with the Workspace

```bash
# From the go/ directory
go work sync          # Sync workspace dependencies
go work use ./path    # Add module to workspace
go work edit -dropuse ./path  # Remove module from workspace
```

## Metrics & Monitoring

The inventory service includes sophisticated metrics collection using goroutines and channels:

### Features
- **Automatic HTTP Metrics**: Request counts, response times, status codes
- **Business Metrics**: Product operations, inventory tracking, validation errors
- **Concurrent Collection**: Non-blocking metrics recording with background processing
- **Thread-Safe**: Proper synchronization with RWMutex and channels
- **Real-time API**: Live metrics via REST endpoints

### Metrics Endpoints
- `GET /metrics` - All collected metrics in JSON format
- `GET /metrics/summary` - Aggregated metrics summary by type
- `POST /metrics/reset` - Reset all metrics counters

### Sample Metrics Response
```json
{
  "http_requests_total": {"value": 150, "labels": {"method": "GET", "path": "/api/v1/products"}},
  "product_operations_total": {"value": 25, "labels": {"operation": "create"}},
  "response_time_ms": {"value": 2.5, "labels": {"endpoint": "/api/v1/products"}}
}
```


## License

Lee@BidOne
