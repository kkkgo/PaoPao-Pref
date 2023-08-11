FROM alpine:edge
RUN apk add  dnscrypt-proxy git bind-tools
WORKDIR /data
COPY check.sh /usr/bin/
RUN chmod +x /usr/bin/check.sh
CMD check.sh