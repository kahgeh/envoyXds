module github.com/kahgeh/envoyXds

go 1.12

require (
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/envoyproxy/go-control-plane v0.7.1
	github.com/envoyproxy/protoc-gen-validate v0.0.14 // indirect
	github.com/gogo/googleapis v1.2.0 // indirect
	github.com/gogo/protobuf v1.2.1
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/swdyh/go-enumerable v0.0.0-20130811053345-498cbb6cfacc
	github.com/thoas/go-funk v0.4.0
	google.golang.org/grpc v1.20.1
)

replace github.com/kahgeh/envoyXds => ./
