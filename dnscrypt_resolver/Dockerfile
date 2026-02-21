FROM golang:alpine AS builder
WORKDIR /app
COPY check.go .
RUN go build -ldflags="-s -w" -o check check.go

FROM alpine:edge
RUN apk add --no-cache dnscrypt-proxy bind-tools ca-certificates tzdata
WORKDIR /data
COPY --from=builder /app/check /usr/bin/check
CMD ["/usr/bin/check"]