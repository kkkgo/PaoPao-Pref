FROM alpine:edge AS builder
COPY . /src/
WORKDIR /src
RUN apk update && apk add go
RUN sh /src/docker-build.sh
FROM alpine:edge
COPY --from=builder /cp/ /data/
RUN mv /data/paopao-pref /usr/bin/
WORKDIR /data/
ENV TZ=Asia/Shanghai \
    DNS_SERVER="" \
    DNS_PORT="" \
    DNS_LINE="" \
    DNS_LIMIT="" \
    DNS_TIMEOUT="" \
    DNS_LOG=""
ENTRYPOINT ["paopao-pref"]