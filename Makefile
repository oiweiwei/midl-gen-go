default: build

.PHONY: gen
gen:
	@goyacc -v midl/y.out -o midl/parse.go -p RPC midl/parse.y

.PHONY: gen-oem
gen-oem:
	@go generate ./...

.PHONY: bin
bin:
	@mkdir -p bin/

.PHONY: build
build: bin gen
	@CGO_ENABLED=0 go build -o bin/midl-gen-go main.go

IMAGE ?= ghcr.io/oiweiwei/midl-gen-go
TAG   ?= latest

.PHONY: docker
docker:
	docker build -t $(IMAGE):$(TAG) .
