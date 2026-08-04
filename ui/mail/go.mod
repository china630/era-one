module era/ui/mail

go 1.25.0

require (
	era/services/platform v0.0.0
	github.com/golang-jwt/jwt/v5 v5.3.1
)

require (
	era/services/license v0.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace era/services/platform => ../../services/platform

replace era/services/license => ../../services/license

replace era/contracts/gen => ../../gen/go
