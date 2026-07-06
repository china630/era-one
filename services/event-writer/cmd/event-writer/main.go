// Command event-writer — Kafka consumer → ClickHouse + API последних событий (S1-5, S1-10).
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"era/services/event-writer/internal/chwriter"
	"era/services/event-writer/internal/consumer"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	chAddr := env("ERA_CH_ADDR", "localhost:9000")
	brokers := env("ERA_KAFKA_BROKERS", "localhost:9092")
	groupID := env("ERA_CONSUMER_GROUP", "era-event-writer")
	httpAddr := env("ERA_HTTP_ADDR", ":8089")

	writer, err := chwriter.New(chAddr)
	if err != nil {
		log.Fatalf("clickhouse: %v", err)
	}
	defer writer.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.URL.Query().Get("node_id")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		rows, err := writer.QueryRecent(r.Context(), nodeID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": rows})
	})
	mux.HandleFunc("/api/timeline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		nodeID := r.URL.Query().Get("node_id")
		corrID := r.URL.Query().Get("correlation_id")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		rows, err := writer.QueryTimeline(r.Context(), nodeID, corrID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"timeline": rows})
	})
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(uiDir()))))

	httpSrv := &http.Server{Addr: httpAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("event-writer HTTP %s (API /api/events, UI /ui/)", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	runner := consumer.New(consumer.ParseBrokers(brokers), groupID, writer)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := runner.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown…")
	cancel()
	runner.Close()
	shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shCancel()
	_ = httpSrv.Shutdown(shCtx)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func uiDir() string {
	if v := os.Getenv("ERA_UI_DIR"); v != "" {
		return v
	}
	// от services/event-writer → repo/ui/events
	return filepath.Join("..", "..", "ui", "events")
}
