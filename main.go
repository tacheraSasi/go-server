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

type Server struct {
    mux          *http.ServeMux
    middleware   []Middleware
    cache        sync.Map
    port         string
    requestCount int
}

// Middleware function type, which can modify HTTP requests or responses.
type Middleware func(http.Handler) http.Handler

func NewServer(port string) *Server {
    return &Server{
        mux:        http.NewServeMux(),
        port:       port,
        middleware: []Middleware{},
    }
}

func (s *Server) AddRoute(path string, handler http.HandlerFunc) {
    finalHandler := handler
    for _, mw := range s.middleware {
        finalHandler = mw(finalHandler)
    }
    s.mux.Handle(path, finalHandler)
}

func (s *Server) AddMiddleware(mw Middleware) {
    s.middleware = append(s.middleware, mw)
}

func (s *Server) Start() {
    srv := &http.Server{
        Addr:         ":" + s.port,
        Handler:      s.mux,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
    }
    log.Printf("Server started on port %s", s.port)
    log.Fatal(srv.ListenAndServe())
}


func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Received %s request for %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func (s *Server) CacheMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if val, ok := s.cache.Load(r.URL.Path); ok {
            log.Printf("Cache hit for %s", r.URL.Path)
            fmt.Fprint(w, val)
            return
        }
        log.Printf("Cache miss for %s", r.URL.Path)
        rw := &responseWriter{ResponseWriter: w}
        next.ServeHTTP(rw, r)
        s.cache.Store(r.URL.Path, rw.Body.String())
    })
}

// Custom response writer to capture response for caching.
type responseWriter struct {
    http.ResponseWriter
    Body *bytes.Buffer
}

func (rw *responseWriter) Write(b []byte) (int, error) {
    if rw.Body == nil {
        rw.Body = &bytes.Buffer{}
    }
    rw.Body.Write(b)
    return rw.ResponseWriter.Write(b)
}


func main() {
    server := NewServer("8080")

    // Adding middleware
    server.AddMiddleware(LoggingMiddleware)
    server.AddMiddleware(server.CacheMiddleware)

    // Adding routes
    server.AddRoute("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Welcome to Go Web Server!")
    })
    server.AddRoute("/static", http.FileServer(http.Dir("./static")).ServeHTTP)

    server.Start()
}


func (s *Server) Start() {
    srv := &http.Server{
        Addr:         ":" + s.port,
        Handler:      s.mux,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Could not listen on %s: %v\n", s.port, err)
        }
    }()

    // Graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt)
    <-stop

    log.Println("Shutting down the server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }
    log.Println("Server exiting")
}


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
