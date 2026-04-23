FROM golang:1.25-alpine AS builder

RUN apk add --no-cache make gcc musl-dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/midl-gen-go main.go


FROM scratch

COPY --from=builder /src/codegen/doc/data /codegen/doc/data

COPY --from=builder /out/midl-gen-go /midl-gen-go

ENTRYPOINT ["/midl-gen-go"]
