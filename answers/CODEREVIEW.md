# Code Review: User Creation HTTP Handler

## Code Under Review
```go
var users = make(map[string]string)

func createUser(name string) {
    users[name] = time.Now().String()
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    go createUser(name)
    w.WriteHeader(http.StatusOK)
}
```

## Executive Summary
This code has **critical concurrency issues** and several design problems.

**Severity**: 🔴 **CRITICAL** - Must be fixed before deployment

## Critical Issues

### 1. 🔴 **CRITICAL: Data Race Condition**
**Issue**: The global `users` map is accessed concurrently without synchronization, causing undefined behavior.

**Risk**:
- Application crashes with "concurrent map writes" panic
- Data corruption and lost updates
- Unpredictable behavior under load

**Fix**: Use thread-safe data structures
```go
type UserStore struct {
    mu    sync.RWMutex
    users map[string]string
}

func NewUserStore() *UserStore {
    return &UserStore{users: make(map[string]string)}
}

func (s *UserStore) CreateUser(name string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.users[name] = time.Now().UTC().Format(time.RFC3339)
}

func (s *UserStore) GetUser(name string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    timestamp, exists := s.users[name]
    return timestamp, exists
}
```

### 2. 🟡 **HIGH: Goroutine Leak Risk**
**Issue**: Unbounded goroutine creation without backpressure control.

**Risk**:
- Memory exhaustion under high load
- CPU thrashing from excessive context switching
- Potential DoS vulnerability

**Fix**: Remove unnecessary goroutine or use worker pool
```go
// Option 1: Remove goroutine (recommended for simple operations)
func handleRequest(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    if name == "" {
        http.Error(w, "missing name parameter", http.StatusBadRequest)
        return
    }

    store.CreateUser(name) // Execute synchronously
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "created",
        "name":   name,
    })
}
```

## Major Issues

### 3. 🟡 **Input Validation Missing**
**Issue**: No validation of the `name` parameter.

**Risk**:
- Empty keys in the map
- Potential injection attacks
- Poor user experience

**Fix**:
```go
name := strings.TrimSpace(r.URL.Query().Get("name"))
if name == "" {
    http.Error(w, `{"error":"name parameter is required"}`, http.StatusBadRequest)
    return
}

// Additional validation
if len(name) > 100 {
    http.Error(w, `{"error":"name too long (max 100 characters)"}`, http.StatusBadRequest)
    return
}

// Sanitize input
if !isValidName(name) {
    http.Error(w, `{"error":"invalid name format"}`, http.StatusBadRequest)
    return
}

func isValidName(name string) bool {
    // Example: only allow alphanumeric and basic punctuation
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9\s\-_.]+$`, name)
    return matched
}
```

### 4. 🟡 **Poor HTTP Response Design**
**Issue**: Always returns 200 OK without meaningful response body.

**Problems**:
- No indication of success/failure
- Missing Content-Type header
- No structured response format

**Fix**:
```go
w.Header().Set("Content-Type", "application/json")

response := map[string]interface{}{
    "status":    "created",
    "name":      name,
    "createdAt": time.Now().UTC().Format(time.RFC3339),
    "id":        generateUserID(), // Consider adding unique ID
}

w.WriteHeader(http.StatusCreated) // 201 for resource creation
json.NewEncoder(w).Encode(response)
```

## Minor Issues

### 5. 🟢 **Time Format Inconsistency**
**Issue**: `time.Now().String()` produces locale-dependent, debug-oriented output.

**Fix**: Use standardized RFC3339 format
```go
timestamp := time.Now().UTC().Format(time.RFC3339)
```

### 6. 🟢 **Global Variable Usage**
**Issue**: Global `users` variable makes testing difficult and violates dependency injection principles.

**Fix**: Use dependency injection
```go
type Handler struct {
    store *UserStore
}

func NewHandler(store *UserStore) *Handler {
    return &Handler{store: store}
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Implementation using h.store
}
```

## Improved HandleCreateUser - Sample only

```go
// HandleCreateUser handles user creation requests
func (h *Handler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
    // Only allow POST requests
    if r.Method != http.MethodPost {
        h.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Get and validate name
    name := strings.TrimSpace(r.URL.Query().Get("name"))
    if err := h.validator.ValidateName(name); err != nil {
        h.sendError(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Check if user already exists
    if _, exists := h.store.GetUser(name); exists {
        h.sendError(w, "user already exists", http.StatusConflict)
        return
    }

    // Create user (synchronously - no unnecessary goroutine)
    user := h.store.CreateUser(name)

    // Send success response
    h.sendSuccess(w, map[string]interface{}{
        "status": "created",
        "user":   user,
    }, http.StatusCreated)
}
```

