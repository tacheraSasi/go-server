package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Server struct - this struct will hold routes, middleware, cache, and configuration.
type Server struct {
	mux          *http.ServeMux
	middleware   []Middleware
	cache        sync.Map
	port         string
}

// Middleware - middleware function type for modifying HTTP requests or responses.
type Middleware func(http.Handler) http.Handler

// NewServer function - initializing a new Server with a specified port.
func NewServer(port string) *Server {
	// Using http.ServeMux to manage routing paths and handlers.
	return &Server{
		mux:        http.NewServeMux(),
		port:       port,
		middleware: []Middleware{}, // Empty middleware list to start.
	}
}

func (s *Server) AddRoute(path string, handler http.HandlerFunc) {
    // So, I want to set up this finalHandler as an http.Handler. Starting with the handler passed in.
    var finalHandler http.Handler = handler 

    // Now, let’s apply each middleware in order
    for _, mw := range s.middleware {
        // Wrapping the handler in middleware layer by layer
        finalHandler = mw(finalHandler)
    }
    
    // Finally, attaching the fully wrapped handler to our ServeMux
    s.mux.Handle(path, finalHandler)
}


// AddMiddleware - Adds middleware to the Server struct’s middleware list.
func (s *Server) AddMiddleware(mw Middleware) {
	s.middleware = append(s.middleware, mw)
}

// Start - Method to start the server with graceful shutdown handling.
func (s *Server) Start() {
	srv := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Launch server in a goroutine to allow graceful shutdown
	go func() {
		log.Printf("Server started on port %s", s.port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", s.port, err)
		}
	}()

	// Setup channel to catch OS signals for graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Println("Shutting down the server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt server shutdown, logging errors if any.
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting")
}

// LoggingMiddleware - Middleware for logging requests.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request for %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// CacheMiddleware - Middleware for caching responses in sync.Map.
func (s *Server) CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if response exists in cache.
		if val, ok := s.cache.Load(r.URL.Path); ok {
			log.Printf("Cache hit for %s", r.URL.Path)
			fmt.Fprint(w, val)
			return
		}

		// If not cached, capture response for future use.
		log.Printf("Cache miss for %s", r.URL.Path)
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		s.cache.Store(r.URL.Path, rw.Body.String()) // Store response in cache.
	})
}

// Custom responseWriter for caching.
type responseWriter struct {
	http.ResponseWriter
	Body *bytes.Buffer
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Initialize buffer if Body is nil.
	if rw.Body == nil {
		rw.Body = &bytes.Buffer{}
	}
	rw.Body.Write(b) // Store written data in buffer for caching.
	return rw.ResponseWriter.Write(b) // Write data to actual response.
}

// main function to set up server with routes and middleware.
func main() {
	server := NewServer("8080")

	// Adding LoggingMiddleware to track each request.
	server.AddMiddleware(LoggingMiddleware)

	// Adding CacheMiddleware to cache responses to repeated requests.
	server.AddMiddleware(server.CacheMiddleware)

	// Adding a simple route to demonstrate response.
	server.AddRoute("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Welcome to Go Web Server!")
	})

	// Static file handling route.
	server.AddRoute("/static", http.FileServer(http.Dir("./static")).ServeHTTP)

	// Start the server.
	server.Start()
}

// StartTLS - Starts the server with TLS, providing HTTP/2 support.
func (s *Server) StartTLS(certFile, keyFile string) {
	srv := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf("Server started with HTTPS on port %s", s.port)
	log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
}
