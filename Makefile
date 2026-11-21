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

# Golden regression check of the generator against a recorded output.
#   make golden-record IDL=../go-msrpc/idl GOLDEN=/path/to/golden
#   make golden        IDL=../go-msrpc/idl GOLDEN=/path/to/golden
# IDL must have msdn/ (index.yaml, extra.yaml, .cache/) as its sibling directory.
IDL    ?= ../go-msrpc/idl
GOLDEN ?= .golden

.PHONY: golden-record
golden-record:
	scripts/golden.sh record $(IDL) $(GOLDEN)

.PHONY: golden
golden:
	scripts/golden.sh check $(IDL) $(GOLDEN)

.PHONY: test
test:
	go test ./...
	cd examples && go test -v ./...

