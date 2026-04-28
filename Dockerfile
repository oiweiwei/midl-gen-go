FROM golang:1.25-alpine AS builder

RUN apk add --no-cache make gcc musl-dev ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/midl-gen-go main.go


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/midl-gen-go /midl-gen-go
COPY --from=builder /src/msdn/index.yaml /msdn/index.yaml
COPY --from=builder /src/msdn/extra.yaml /msdn/extra.yaml

ENTRYPOINT ["/midl-gen-go"]
