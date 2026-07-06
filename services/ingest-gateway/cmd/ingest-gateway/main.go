// Command ingest-gateway — приём телеметрии ERA XDR (HTTP + gRPC → Kafka).
package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	erav1 "era/contracts/gen/era/v1"
	"era/services/ingest-gateway/internal/grpcserver"
	"era/services/ingest-gateway/internal/kafka"
	"era/services/ingest-gateway/internal/server"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
	"era/services/platform/tlsutil"
	"google.golang.org/grpc"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	httpAddr := env("ERA_HTTP_ADDR", ":8082")
	grpcAddr := env("ERA_GRPC_ADDR", ":50051")
	brokers := strings.Split(env("ERA_KAFKA_BROKERS", "localhost:9092"), ",")

	producer := kafka.NewProducer(brokers)
	defer producer.Close()

	if err := licensegate.ValidateStartup(0); err != nil {
		log.Fatalf("license: %v", err)
	}

	ingestGRPC := grpcserver.New(producer)

	grpcOpts := []grpc.ServerOption{}
	if tlsCfg := tlsutil.ServerFromEnv(); tlsCfg.Enabled() {
		creds, err := tlsCfg.Load()
		if err != nil {
			log.Fatalf("tls: %v", err)
		}
		grpcOpts = append(grpcOpts, grpc.Creds(creds))
		log.Printf("ingest-gateway gRPC mTLS enabled")
	}
	grpcSrv := grpc.NewServer(grpcOpts...)
	erav1.RegisterIngestServiceServer(grpcSrv, ingestGRPC)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	go func() {
		log.Printf("ingest-gateway gRPC слушает %s", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// HTTP (TLS via httpserver when ERA_TLS_* set)
	httpHandler := server.Routes(server.Config{Producer: producer, GRPC: ingestGRPC})
	go func() {
		log.Printf("ingest-gateway HTTP слушает %s", httpAddr)
		if err := httpserver.Listen(httpAddr, httpHandler); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("получен сигнал остановки, graceful shutdown…")

	grpcSrv.GracefulStop()
	log.Println("ingest-gateway остановлен")
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
