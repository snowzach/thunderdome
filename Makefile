EXECUTABLE := tdome
GITVERSION := $(shell git describe --dirty --always --tags --long)
GOPATH ?= ${HOME}/go
PACKAGENAME := $(shell go list -m -f '{{.Path}}')
EMBEDDIR := embed
MIGRATIONDIR := store/postgres/migrations
MIGRATIONS :=  $(wildcard ${MIGRATIONDIR}/*.sql)
TOOLS := ${GOPATH}/bin/go-bindata \
	${GOPATH}/bin/mockery \
	${GOPATH}/src/github.com/gogo/protobuf/proto \
	${GOPATH}/bin/protoc-gen-gogoslick \
	${GOPATH}/bin/protoc-gen-grpc-gateway \
	${GOPATH}/bin/protoc-gen-swagger \
	${GOPATH}/bin/wire
export PROTOBUF_INCLUDES = -I. -I/usr/include -I${GOPATH}/src -I$(shell go list -e -f '{{.Dir}}' .) -I$(shell go list -e -f '{{.Dir}}' github.com/lightningnetwork/lnd) -I$(shell go list -e -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/runtime)/../ -I$(shell go list -e -f '{{.Dir}}' github.com/grpc-ecosystem/grpc-gateway/runtime)/../third_party/googleapis
PROTOS := ./server/versionrpc/version.pb.gw.go \
	./tdrpc/tdrpc.pb.gw.go \
	./tdrpc/adminrpc.pb.gw.go
SWAGGERDOCS = ./server/versionrpc/version.swagger.json \
	./tdrpc/tdrpc.swagger.json \
	./tdrpc/adminrpc.swagger.json
SWAGGER_VERSION = 3.20.8

.PHONY: default
default: ${EXECUTABLE}

# This is all the tools required to compile, test and handle protobufs
tools: ${TOOLS}

${GOPATH}/bin/go-bindata:
	GO111MODULE=off go get -u github.com/go-bindata/go-bindata/...

${GOPATH}/bin/mockery:
	go get github.com/vektra/mockery/cmd/mockery

${GOPATH}/src/github.com/gogo/protobuf/proto:
	GO111MODULE=off go get github.com/gogo/protobuf/proto

${GOPATH}/bin/protoc-gen-gogoslick:
	go get github.com/gogo/protobuf/protoc-gen-gogoslick

${GOPATH}/bin/protoc-gen-grpc-gateway:
	go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

${GOPATH}/bin/protoc-gen-swagger:
	go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

${GOPATH}/bin/wire:
	go get github.com/google/wire

# Handle all grpc endpoint protobufs
%.pb.gw.go: %.proto
	protoc ${PROTOBUF_INCLUDES} --gogoslick_out=paths=source_relative,plugins=grpc,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types:. --grpc-gateway_out=paths=source_relative,allow_colon_final_segments=true,logtostderr=true:. --swagger_out=logtostderr=true:. $*.proto

# Handle any non-specific protobufs
%.pb.go: %.proto
	protoc ${PROTOBUF_INCLUDES} --gogoslick_out=paths=source_relative,plugins=grpc:. $*.proto

${EMBEDDIR}/bindata.go: ${SWAGGERDOCS} embed/public/api-docs/index.html embed/public/swagger-ui/index.html
	# Copying swagger docs
	mkdir -p embed/public/api-docs
	cp $(SWAGGERDOCS) embed/public/api-docs
	# Building bindata
	go-bindata -o ${EMBEDDIR}/bindata.go -prefix ${EMBEDDIR} -pkg embed embed/public/...

${MIGRATIONDIR}/bindata.go: ${MIGRATIONS}
	# Building bindata
	go-bindata -o ${MIGRATIONDIR}/bindata.go -prefix ${MIGRATIONDIR} -pkg migrations ${MIGRATIONDIR}/*.sql

cmd/wire_gen.go: cmd/wire.go
	wire ./cmd/...

.PHONY: mocks
mocks: tools
	mockery -dir ./tdrpc -name Store
	mockery -dir ./store -name DistCache
	mockery -dir $(shell go list -e -f '{{.Dir}}' github.com/go-redis/redis) -name UniversalClient
	mockery -dir $(shell go list -e -f '{{.Dir}}' github.com/lightningnetwork/lnd/lnrpc) -name LightningClient

.PHONY: ${EXECUTABLE}
${EXECUTABLE}: tools ${PROTOS} ${MIGRATIONDIR}/bindata.go ${EMBEDDIR}/bindata.go cmd/wire_gen.go
	# Compiling...
	go build -ldflags "-X ${PACKAGENAME}/conf.Executable=${EXECUTABLE} -X ${PACKAGENAME}/conf.GitVersion=${GITVERSION}" -o ${EXECUTABLE}

.PHONY: test
test: tools ${PROTOS} ${MIGRATIONDIR}/bindata.go ${EMBEDDIR}/bindata.go cmd/wire_gen.go mocks
	go test -cover ./...

.PHONY: golangci-lint
golangci-lint: tools ${PROTOS} ${MIGRATIONDIR}/bindata.go ${EMBEDDIR}/bindata.go cmd/wire_gen.go mocks
	golangci-lint run

.PHONY: deps
deps:
	# Fetching dependancies...
	go get -d -v # Adding -u here will break CI

embed/public/swagger-ui/index.html:
	# Downloading Swagger UI
	mkdir -p embed/public/swagger-ui
	curl -L https://github.com/swagger-api/swagger-ui/archive/v${SWAGGER_VERSION}.tar.gz | tar zx --strip-components 2 -C embed/public/swagger-ui swagger-ui-${SWAGGER_VERSION}/dist
