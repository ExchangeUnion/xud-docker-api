FROM golang:1.15-alpine3.12 as builder
RUN apk --no-cache add make
WORKDIR github.com/ExchangeUnion/xud-docker-api
COPY go.mod .
COPY go.sum .
RUN go mod download
ADD . .
RUN make

FROM alpine:3.12
RUN apk add --no-cache bash docker-cli
COPY --from=builder /go/github.com/ExchangeUnion/xud-docker-api/proxy /usr/local/bin/proxy
ENTRYPOINT ["proxy"]
