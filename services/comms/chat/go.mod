module era/services/comms/chat

go 1.25.0

require (
	era/services/comms/internal/httpauth v0.0.0
	era/services/platform v0.0.0
	github.com/jackc/pgx/v5 v5.10.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace (
	era/contracts/gen => ../../../gen/go
	era/services/comms/internal/httpauth => ../internal/httpauth
	era/services/platform => ../../platform
)

replace era/services/license => ../../license
