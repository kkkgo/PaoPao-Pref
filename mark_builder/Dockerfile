FROM sliamb/paopaodns AS download
COPY --from=sliamb/paopao-pref /usr/bin/paopao-pref /usr/bin/
WORKDIR /data
COPY download.sh /data/
RUN sh /data/download.sh

FROM alpine:edge
RUN apk add --no-cache curl xz
COPY --from=sliamb/paopao-pref /usr/bin/paopao-pref /usr/bin/
COPY --from=sliamb/prebuild-paopaodns /src/mosdns /usr/bin/
COPY --from=download /src/Country-only-cn-private.mmdb /data/
COPY --from=download /data/inrule.txt /data/
COPY --from=download /data/domains.txt /
COPY dns_list.txt /data/
COPY test_cn.yaml /data/
COPY mark_update.sh /data/
ENV TZ=Asia/Shanghai \
    DNS_SERVER="" \
    DNS_PORT="" \
    DNS_LINE="" \
    DNS_LIMIT="25" \
    DNS_TIMEOUT="3s" \
    DNS_SLEEP="0ms" \
    FILE_OUTPUT=yes
WORKDIR /data/
CMD sh mark_update.sh