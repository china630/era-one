module era/services/vm

go 1.22.7

require (
	era/contracts/gen v0.0.0
	github.com/google/uuid v1.6.0
	github.com/oklog/ulid v1.3.1
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/protobuf v1.35.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.68.0 // indirect
)

replace era/contracts/gen => ../../gen/go
