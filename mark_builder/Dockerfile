FROM alpine:edge
RUN apk add --no-cache curl xz
COPY --from=sliamb/paopao-pref /usr/bin/paopao-pref /usr/bin/
COPY --from=sliamb/prebuild-paopaodns /src/mosdns /usr/bin/
ADD https://github.com/kkkgo/PaoPao-Pref/raw/main/domains.txt /data/
COPY dns_list.txt /data/
COPY test_cn.yaml /data/
COPY mark_update.sh /data/
ENV TZ=Asia/Shanghai \
    DNS_SERVER="" \
    DNS_PORT="" \
    DNS_LINE="" \
    DNS_LIMIT="" \
    DNS_TIMEOUT="" \
    DNS_SLEEP="" \
    DNS_LOG=""
WORKDIR /data/
CMD sh mark_update.sh 