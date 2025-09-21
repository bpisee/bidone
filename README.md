# BidOne - Inventory Service

A high-performance inventory management service built with Go and Echo framework, featuring comprehensive metrics collection using goroutines and channels.

## Quick Start

### Prerequisites
- Go 1.19 or later
- Git

### Installation & Setup

```bash
# Clone the repository
git clone <repository-url>
cd bidone

# Install dependencies
go mod tidy
```

## Running the Inventory Service

### Build and Run
```bash
# Build the service
go build -o bin/inventorysvc.exe ./cmd/inventorysvc/

# Run the service
./bin/inventorysvc.exe
```

The service will start on `http://localhost:8080` with the following endpoints:

### API Endpoints
- **Health Check**: `GET /health`
- **Products API**: `GET|POST|PUT|DELETE /api/v1/products`
- **Metrics API**: `GET /metrics`, `GET /metrics/summary`, `POST /metrics/reset`

### Example API Usage

```bash
# Health check
curl http://localhost:8080/health

# Create a product
curl -X POST http://localhost:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Product","price_cents":2999,"quantity":10}'

# List products
curl http://localhost:8080/api/v1/products

# Get metrics summary
curl http://localhost:8080/metrics/summary
```

## Testing

### Run All Tests
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test -v ./cmd/inventorysvc/
go test -v ./pkg/metrics/
go test -v ./internal/services/
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
├── cmd/inventorysvc/          # HTTP server and handlers
├── internal/
│   ├── services/              # Business logic layer
│   └── store/                 # Data access layer
├── pkg/
│   ├── metrics/               # Metrics collection
│   └── model/                 # Data models
└── bin/                       # Built binaries
```

## Development

### Code Quality
```bash
# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run

# Vet code
go vet ./...
```

### Building for Production
```bash
# Build optimized binary
go build -ldflags="-s -w" -o bin/inventorysvc ./cmd/inventorysvc/
```

## Metrics

The service includes a sophisticated metrics collection using goroutines and channels:

- **Automatic HTTP Metrics**: Request counts, response times, status codes
- **Business Metrics**: Product operations, inventory tracking, validation errors
- **Concurrent Collection**: Non-blocking metrics recording with background processing
- **Thread-Safe**: Proper synchronization with RWMutex and channels
- **API Access**: Real-time metrics via REST endpoints

### Metrics Endpoints
- `GET /metrics` - All collected metrics
- `GET /metrics/summary` - Metrics summary by type
- `POST /metrics/reset` - Reset all metrics

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

Lee@BidOne
