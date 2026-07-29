module era/services/comms/mail-connect

go 1.25.0

require era/services/platform v0.0.0

require (
	github.com/klauspost/compress v1.18.6 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace era/services/platform => ../../platform

replace era/services/license => ../../license
