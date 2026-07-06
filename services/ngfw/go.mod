module era/services/ngfw

go 1.22.7

require era/services/platform v0.0.0

require (
	era/contracts/gen v0.0.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.68.0 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
)

replace (
	era/contracts/gen => ../../gen/go
	era/services/platform => ../platform
)
