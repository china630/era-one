module era/services/comms/mail

go 1.25.0

require (
	era/contracts/gen v0.0.0
	era/services/comms/calendar v0.0.0
	era/services/comms/internal/httpauth v0.0.0
	era/services/platform v0.0.0
	github.com/ClickHouse/clickhouse-go/v2 v2.47.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/minio/minio-go/v7 v7.2.1
	github.com/oklog/ulid v1.3.1
	golang.org/x/crypto v0.53.0
	google.golang.org/protobuf v1.36.11
)

require (
	era/services/license v0.0.0 // indirect
	github.com/ClickHouse/ch-go v0.73.0 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/paulmach/orb v0.13.0 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/grpc v1.82.1 // indirect
	gopkg.in/ini.v1 v1.67.2 // indirect
)

exclude google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd

exclude google.golang.org/genproto v0.0.0-20200825200019-8632dd797987

exclude google.golang.org/genproto v0.0.0-20220822174746-9e6da59bd2fc

// Workspace-wide pin (go.work applies replaces to all modules). Keep ≥ fixed GO-2026-6061.
replace google.golang.org/grpc => google.golang.org/grpc v1.82.1

replace (
	era/contracts/gen => ../../../gen/go
	era/services/comms/calendar => ../calendar
	era/services/comms/internal/httpauth => ../internal/httpauth
	era/services/platform => ../../platform
)

replace era/services/license => ../../license
