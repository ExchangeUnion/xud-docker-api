FROM golang:1.15.2-alpine3.12 as builder
WORKDIR github.com/ExchangeUnion/xud-docker-api-poc
COPY go.mod .
COPY go.sum .
RUN go mod download
ADD . .
RUN go build ./cmd/proxy

FROM alpine:3.12
COPY --from=builder /go/github.com/ExchangeUnion/xud-docker-api-poc/proxy /usr/local/bin/proxy
ENTRYPOINT ["proxy"]
