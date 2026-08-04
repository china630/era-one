module era/services/ingest-gateway

go 1.25.0

require (
	era/contracts/gen v0.0.0
	era/services/platform v0.0.0
	github.com/oklog/ulid v1.3.1
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

require (
	era/services/license v0.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
)

replace era/contracts/gen => ../../gen/go

replace era/services/platform => ../platform

replace era/services/license => ../license
