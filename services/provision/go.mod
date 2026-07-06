module era/services/provision

go 1.22.7

require (
	era/services/platform v0.0.0
	github.com/minio/minio-go/v7 v7.0.80
)

replace era/services/platform => ../platform
