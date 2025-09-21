var users = make(map[string]string)
func createUser(name string) {
users[name] = time.Now().String()
}
func handleRequest(w http.ResponseWriter, r *http.Request) {
name := r.URL.Query().Get("name")
go createUser(name)
w.WriteHeader(http.StatusOK)
}
