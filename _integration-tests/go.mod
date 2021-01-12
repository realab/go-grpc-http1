module github.com/realab/go-grpc-http1/_integration-tests

go 1.14

require (
	github.com/realab/go-grpc-http1 v0.0.0+incompatible
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	google.golang.org/grpc v1.27.1
)

replace github.com/realab/go-grpc-http1 => ../
