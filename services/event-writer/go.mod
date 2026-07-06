module era/services/event-writer

go 1.22.7

require (
	era/contracts/gen v0.0.0
	era/services/platform v0.0.0
	github.com/ClickHouse/clickhouse-go/v2 v2.30.0
	github.com/oklog/ulid v1.3.1
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/protobuf v1.35.1
)

require (
	github.com/ClickHouse/ch-go v0.61.5 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/paulmach/orb v0.11.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	go.opentelemetry.io/otel v1.26.0 // indirect
	go.opentelemetry.io/otel/trace v1.26.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.68.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	era/contracts/gen => ../../gen/go
	era/services/license => ../license
	era/services/platform => ../platform
)
