FROM alpine:edge AS builder
COPY . /src/
WORKDIR /src
RUN apk update && apk add go
RUN go build -ldflags "-s -w" -trimpath -o /paopao-pref
FROM alpine:edge
COPY --from=builder /paopao-pref /usr/bin/
ADD https://github.com/kkkgo/PaoPao-Pref/raw/main/domains.txt /data/
WORKDIR /data/
ENV TZ=Asia/Shanghai \
    DNS_SERVER="" \
    DNS_PORT="" \
    DNS_LINE="" \
    DNS_LIMIT="" \
    DNS_TIMEOUT="" \
    DNS_LOG=""
ENTRYPOINT ["paopao-pref"]