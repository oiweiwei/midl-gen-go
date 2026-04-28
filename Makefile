default: build

.PHONY: gen
gen:
	@go generate ./...
	@goyacc -v midl/y.out -o midl/parse.go -p RPC midl/parse.y

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

.PHONY: demo
demo:
	./bin/midl-gen-go generate \
		--pkg github.com/oiweiwei/midl-gen-go/examples/demo/ \
		-o examples/demo \
		-I ../go-msrpc/idl/ \
		-I examples/demo/idl/ \
			examples/demo/idl/**

.PHONY: test-demo
test-demo:
	cd examples && go test -v ./...

.PHONY:
test:
	go test ./...
