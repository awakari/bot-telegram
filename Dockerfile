FROM golang:1.24.3-alpine3.21 AS builder
WORKDIR /go/src/bot-telegram
COPY . .
RUN \
    apk add protoc protobuf-dev make git && \
    make build

FROM scratch
COPY --from=builder /go/src/bot-telegram/bot-telegram /bin/bot-telegram
ENTRYPOINT ["/bin/bot-telegram"]
