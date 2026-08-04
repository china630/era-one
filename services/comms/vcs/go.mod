module era/services/comms/vcs

go 1.25.0

require (
	era/services/comms/internal/httpauth v0.0.0
	era/services/platform v0.0.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	golang.org/x/net v0.56.0 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace (
	era/contracts/gen => ../../../gen/go
	era/services/comms/internal/httpauth => ../internal/httpauth
	era/services/platform => ../../platform
)

replace era/services/license => ../../license
