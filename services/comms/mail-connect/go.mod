module era/services/comms/mail-connect

go 1.25.0

require era/services/platform v0.0.0

require (
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	golang.org/x/net v0.56.0 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace era/services/platform => ../../platform

replace era/services/license => ../../license

require (
	era/services/comms/internal/httpauth v0.0.0
	era/services/comms/internal/imapclient v0.0.0
)

replace era/services/comms/internal/imapclient => ../internal/imapclient

replace era/services/comms/internal/httpauth => ../internal/httpauth
