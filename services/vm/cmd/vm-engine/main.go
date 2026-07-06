package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"era/services/vm/internal/api"
	"era/services/vm/internal/parser"
	"era/services/vm/internal/publisher"
	"era/services/vm/internal/scanner"
	"era/services/vm/internal/scheduler"
)

const (
	templatesDir = "./templates"
	listenAddr   = ":8081"
	concurrency  = 20
	shutdownWait = 10 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	if err := ensureTemplatesDir(templatesDir); err != nil {
		log.Fatal(err)
	}

	templates, err := parser.LoadTemplatesFromDir(templatesDir)
	if err != nil {
		log.Fatalf("загрузка шаблонов: %v", err)
	}
	if len(templates) == 0 {
		log.Fatal("папка templates пуста: нет ни одного валидного .yaml шаблона")
	}

	exec := scanner.NewHTTPExecutor()
	engine := scanner.NewEngine(exec, templates, concurrency)

	var pub *publisher.Publisher
	if brokers := os.Getenv("ERA_KAFKA_BROKERS"); brokers != "" {
		pub = publisher.New(strings.Split(brokers, ","), envOr("ERA_TENANT_ID", "tenant-dev"), envOr("ERA_NODE_ID", "vm-scanner"))
		defer pub.Close()
		if os.Getenv("ERA_VM_PUBLISH_SMOKE") == "1" {
			if err := pub.PublishSmoke(context.Background()); err != nil {
				log.Printf("vm smoke publish: %v", err)
			} else {
				log.Println("vm smoke finding published to Kafka")
			}
		}
	}

	router := api.SetupRoutes(engine, pub, scheduler.New())

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	go func() {
		log.Printf("HTTP-сервер слушает %s", listenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("получен сигнал остановки, graceful shutdown…")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownWait)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown: %v", err)
	}
	log.Println("сервер остановлен")
}

// ensureTemplatesDir проверяет, что каталог существует и это именно директория.
func ensureTemplatesDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("каталог ./templates не найден")
		}
		return err
	}
	if !info.IsDir() {
		return errors.New("./templates существует, но это не каталог")
	}
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
