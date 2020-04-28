FROM golang:1.13-alpine AS builder

COPY source /app
RUN cd /app \
    && go build -v \
    -o main \
    -tags netgo \
    -ldflags '-extldflags "-static"'

FROM alpine:3.11

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/main /app

ENTRYPOINT ["/app"]
