package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:embed frontend/*
var embeddedFrontend embed.FS

// Telemetry instrumentation metrics
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yolo_api_http_requests_total",
			Help: "Total number of HTTP requests processed, labeled by status code, HTTP method, and request path.",
		},
		[]string{"code", "method", "path"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "yolo_api_http_request_duration_seconds",
			Help:    "Histogram of request processing latencies, in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type App struct {
	DB *sql.DB
}

// loggingResponseWriter captures the status code written to the response writer
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

type contextKey string

const requestIDKey contextKey = "request_id"

// generateRequestID produces a 24-character hexadecimal unique identifier
func generateRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// correlationMiddleware assigns a unique X-Request-ID if missing and injects it into context
func correlationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// telemetryMiddleware intercepts requests to gather Prometheus metrics and emit structured logs
func telemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Normalize paths to prevent high-cardinality label values (e.g. mapping /api/tasks/123 to /api/tasks/:id)
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/tasks/") && len(strings.Split(path, "/")) >= 4 {
			path = "/api/tasks/:id"
		}

		lrw := newLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)
		
		duration := time.Since(start).Seconds()
		statusCodeStr := strconv.Itoa(lrw.statusCode)

		httpRequestsTotal.WithLabelValues(statusCodeStr, r.Method, path).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)

		// Structured JSON logging for cloud-native observability
		reqID, _ := r.Context().Value(requestIDKey).(string)
		logLevel := "INFO"
		if lrw.statusCode >= 500 {
			logLevel = "ERROR"
		} else if lrw.statusCode >= 400 {
			logLevel = "WARN"
		}

		logPayload, _ := json.Marshal(map[string]interface{}{
			"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
			"level":       logLevel,
			"request_id":  reqID,
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      lrw.statusCode,
			"duration_ms": float64(time.Since(start).Microseconds()) / 1000.0,
			"remote_addr": r.RemoteAddr,
		})
		log.Println(string(logPayload))
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:postgres@localhost:5432/yolo?sslmode=disable"
	}

	app := &App{}

	// DevOps Best Practice: Implement retry logic for database connection to handle startup latency
	var db *sql.DB
	var err error
	for i := 1; i <= 5; i++ {
		log.Printf("Connecting to database (attempt %d/5)...", i)
		db, err = sql.Open("postgres", dbDSN)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Println("Successfully connected to database")
				break
			}
		}
		log.Printf("Failed to connect to database: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("Could not connect to database after 5 attempts: %v", err)
	}
	// DevOps SRE Best Practice: Tune database connection pool parameters
	// to avoid socket exhaustion under high load and recycle stale idle connections
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	defer db.Close()
	app.DB = db

	mux := http.NewServeMux()
	
	// Expose standard Prometheus scrape endpoint
	mux.Handle("/metrics", promhttp.Handler())
	
	mux.Handle("/health", telemetryMiddleware(http.HandlerFunc(app.handleHealth)))
	mux.Handle("/livez", http.HandlerFunc(app.handleLivez))
	mux.Handle("/healthz", http.HandlerFunc(app.handleLivez))
	mux.Handle("/readyz", telemetryMiddleware(http.HandlerFunc(app.handleReadyz)))
	mux.Handle("/api/tasks", telemetryMiddleware(http.HandlerFunc(app.handleTasks)))
	mux.Handle("/api/tasks/", telemetryMiddleware(http.HandlerFunc(app.handleTasksWithID)))

	// Serve embedded web dashboard frontend at root
	frontendFS, err := fs.Sub(embeddedFrontend, "frontend")
	if err == nil {
		mux.Handle("/", http.FileServer(http.FS(frontendFS)))
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      correlationMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server listening on port %s", port)
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}

	case sig := <-shutdown:
		log.Printf("Shutdown signal %v received: initiating graceful termination...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v. Forcing shutdown...", err)
			_ = srv.Close()
		}
		log.Println("Server successfully shutdown")
	}
}

func (app *App) handleLivez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"uptime":    "ok",
	})
}

func (app *App) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var dbStatus string
	var errDetails string
	statusCode := http.StatusOK

	if app.DB == nil {
		statusCode = http.StatusServiceUnavailable
		dbStatus = "uninitialized"
	} else if err := app.DB.PingContext(ctx); err != nil {
		statusCode = http.StatusServiceUnavailable
		dbStatus = "disconnected"
		errDetails = err.Error()
	} else {
		dbStatus = "connected"
	}

	w.WriteHeader(statusCode)
	resp := map[string]interface{}{
		"status":    "ready",
		"database":  dbStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if statusCode != http.StatusOK {
		resp["status"] = "not_ready"
		if errDetails != "" {
			resp["error"] = errDetails
		}
	} else if app.DB != nil {
		stats := app.DB.Stats()
		resp["db_pool"] = map[string]int{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
		}
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	err := app.DB.Ping()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "unhealthy",
			"database":    "disconnected",
			"error_details": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "healthy",
		"database": "connected",
		"uptime":   "ok",
	})
}

func (app *App) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		rows, err := app.DB.QueryContext(ctx, "SELECT id, title, description, status, created_at, updated_at FROM tasks ORDER BY id DESC")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tasks := []Task{}
		for rows.Next() {
			var t Task
			if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			tasks = append(tasks, t)
		}

		json.NewEncoder(w).Encode(tasks)

	case http.MethodPost:
		var t Task
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, "Invalid input payload", http.StatusBadRequest)
			return
		}
		if t.Title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}
		if t.Status == "" {
			t.Status = "TODO"
		}

		query := "INSERT INTO tasks (title, description, status) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at"
		err := app.DB.QueryRowContext(ctx, query, t.Title, t.Description, t.Status).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(t)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *App) handleTasksWithID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid task ID path", http.StatusBadRequest)
		return
	}
	idStr := parts[3]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodDelete:
		result, err := app.DB.ExecContext(ctx, "DELETE FROM tasks WHERE id = $1", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if rowsAffected == 0 {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Task %d deleted successfully", id)})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
