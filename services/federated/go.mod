module era/services/federated

go 1.25.0

require (
	era/services/platform v0.0.0
	modernc.org/sqlite v1.34.5
)

require (
	era/contracts/gen v0.0.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/pprof v0.0.0-20260115054156-294ebfa9ad83 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
)

replace (
	era/contracts/gen => ../../gen/go
	era/services/platform => ../platform
)
