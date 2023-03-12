package interfaces

// HTTPServer represents HTTP server
type HTTPServer interface {
	StartHTTPServer()
	CloseHTTPServer()
}
