package main

import (
	"context"
	"log"
	"os"

	"era/services/control-plane/internal/api"
	"era/services/control-plane/internal/hybrid"
	"era/services/control-plane/internal/inventory"
	"era/services/control-plane/internal/store"
	"era/services/platform/licensegate"
	"era/services/platform/tlsutil"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8090")

	st, err := store.NewFromEnv()
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer func() { _ = st.Close() }()
	gate, err := licensegate.GateFromEnv(0)
	if err != nil {
		log.Fatalf("license: %v", err)
	}
	srv := api.New(st, gate)

	if os.Getenv("ERA_STORE_PATH") == "" && os.Getenv("ERA_STORE_DRIVER") == "" {
		st.UpsertAsset(&store.Asset{
			NodeID: "node-01", TenantID: "tenant-dev", Hostname: "win-host-01",
			Platform: "windows", AgentID: "agent-0001", AgentVersion: "0.1.0",
		})
	}

	relay, err := hybrid.NewRelay(st)
	if err != nil {
		log.Fatalf("hybrid relay: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	relay.Start(ctx)

	if brokers := os.Getenv("ERA_KAFKA_BROKERS"); brokers != "" && os.Getenv("ERA_INVENTORY_CONSUMER") != "0" {
		cons := inventory.NewConsumer(inventory.ParseBrokers(brokers), env("ERA_INVENTORY_GROUP", "era-control-plane-inventory"), st)
		go func() {
			if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("inventory consumer: %v", err)
			}
		}()
		defer cons.Close()
		log.Printf("inventory consumer enabled (xdr.inventory)")
	}

	tlsCfg := tlsutil.ServerFromEnv()
	httpSrv := tlsCfg.HTTPServer(addr, srv.Routes())
	if tlsCfg.Enabled() {
		log.Printf("control-plane слушает %s (TLS)", addr)
	} else {
		log.Printf("control-plane слушает %s", addr)
	}
	log.Fatal(tlsCfg.Listen(httpSrv))
}
func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
