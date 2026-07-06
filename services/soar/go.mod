module era/services/soar

go 1.22.7

require (
	era/services/platform v0.0.0
	github.com/google/uuid v1.6.0
)

require era/services/license v0.0.0 // indirect

replace era/services/platform => ../platform

replace era/services/license => ../license
