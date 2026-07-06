module era/services/cloud-portal

go 1.22.7

require (
	era/services/license v0.0.0
	era/services/platform v0.0.0
	github.com/google/uuid v1.6.0
)

require (
	google.golang.org/grpc v1.68.0 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
)

replace era/services/license => ../license

replace era/services/platform => ../platform
