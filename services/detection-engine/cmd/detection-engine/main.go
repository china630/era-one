package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	erav1 "era/contracts/gen/era/v1"
	"era/services/detection-engine/internal/chwriter"
	"era/services/detection-engine/internal/api"
	"era/services/detection-engine/internal/processor"
	"era/services/detection-engine/internal/sigma"
	"era/services/detection-engine/internal/tip"
	"era/services/platform/cpclient"
	"era/services/platform/httpserver"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	brokers := strings.Split(env("ERA_KAFKA_BROKERS", "localhost:9092"), ",")
	corpusPaths := []string{
		env("ERA_SIGMA_CORPUS", filepath.Join("..", "..", "data", "sigma-corpus", "rules")),
		env("ERA_SIGMA_CURATED", filepath.Join("..", "..", "data", "sigma-corpus", "curated")),
	}
	chAddr := env("ERA_CH_ADDR", "localhost:9000")

	var rules []*sigma.Rule
	for _, corpus := range corpusPaths {
		r, err := sigma.LoadDir(corpus)
		if err != nil {
			log.Printf("sigma skip %s: %v", corpus, err)
			continue
		}
		rules = append(rules, r...)
	}
	if len(rules) == 0 {
		log.Fatal("no sigma rules loaded")
	}
	if errs := sigma.Lint(rules); len(errs) > 0 {
		log.Fatalf("sigma lint: %d errors, first: %s", len(errs), errs[0])
	}
	log.Printf("loaded %d sigma rules from %v", len(rules), corpusPaths)

	dw, err := chwriter.New(chAddr, env("ERA_CH_USER", "era"), env("ERA_CH_PASSWORD", "era_dev_pw"))
	if err != nil {
		log.Fatalf("clickhouse: %v", err)
	}
	nationalPath := env("ERA_NATIONAL_IOCS", filepath.Join("..", "..", "data", "national-iocs", "patterns.json"))
	nationalFeed, err := tip.LoadFile(nationalPath)
	if err != nil {
		log.Printf("national IOC feed: %v (continuing without)", err)
		nationalFeed = tip.FromPatterns(nil)
	} else {
		log.Printf("national IOC patterns: %d", nationalFeed.PatternCount())
	}

	stixPath := env("ERA_STIX_BUNDLE", filepath.Join("..", "..", "data", "tip", "bundle.json"))
	stixFeed, err := tip.LoadSTIXBundle(stixPath)
	if err != nil {
		log.Printf("STIX bundle: %v (continuing without)", err)
		stixFeed = tip.FromPatterns(nil)
	} else {
		log.Printf("STIX IOC patterns: %d", stixFeed.PatternCount())
	}

	cpURL := env("ERA_CONTROL_PLANE_URL", "")
	var cp *cpclient.Client
	if cpURL != "" && env("ERA_DETECTION_AUTO_CASE", "1") != "0" {
		cp = cpclient.New(cpURL).WithActor("detection-engine")
		log.Printf("auto-case enabled → %s", cpURL)
	}
	proc := processor.New(rules, dw, nationalFeed, stixFeed, cp)

	groupID := env("ERA_CONSUMER_GROUP", "era-detection-engine")
	httpAddr := env("ERA_HTTP_ADDR", ":8097")
	expSrv := &api.ExposureServer{CH: dw, CP: cp}
	go func() {
		log.Printf("detection-engine HTTP %s (/api/v1/exposure, /metrics)", httpAddr)
		if err := httpserver.Listen(httpAddr, expSrv.Routes()); err != nil {
			log.Printf("http server: %v", err)
		}
	}()

	topics := []string{"xdr.process", "xdr.network", "xdr.auth", "xdr.file", "xdr.dns"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, topic := range topics {
		topic := topic
		go func() {
			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers: brokers, GroupID: groupID, Topic: topic,
			})
			defer r.Close()
			for {
				msg, err := r.ReadMessage(ctx)
				if err != nil {
					return
				}
				var env erav1.Envelope
				if err := proto.Unmarshal(msg.Value, &env); err != nil {
					continue
				}
				proc.Handle(ctx, &env)
			}
		}()
	}

	log.Printf("detection-engine running (%d rules, topics=%v)", len(rules), topics)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
