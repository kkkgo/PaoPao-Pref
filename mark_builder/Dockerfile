FROM sliamb/paopaodns AS download
WORKDIR /src
RUN curl -sLo /src/Country-only-cn-private.mmdb https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb &&\
    mmdb_hash=$(sha256sum /src/Country-only-cn-private.mmdb | grep -Eo "[a-zA-Z0-9]{64}" | head -1) &&\
    mmdb_down_hash=$(curl -s https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb.sha256sum | grep -Eo "[a-zA-Z0-9]{64}" | head -1) &&\
    if [ "$mmdb_down_hash" != "$mmdb_hash" ]; then cp /mmdb_down_hash_error .;exit;fi
RUN curl -sLo /src/inrule_base64.txt https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt &&\
    domains_size=$(wc -c <"/src/inrule_base64.txt") &&\
    if [ "$domains_size" -gt 100000 ]; then echo "domains_size pass.";else echo "domains_size failed";cp /domains_size /;exit;fi &&\
    base64 -d /src/inrule_base64.txt > /src/inrule.txt
RUN curl -sLo /src/domains.txt https://github.com/kkkgo/PaoPao-Pref/raw/main/domains.txt &&\
    domains_size=$(wc -c <"/src/domains.txt") &&\
    if [ "$domains_size" -gt 10000000 ]; then echo "domains_size pass.";else echo "domains_size failed";cp /domains_size /;exit;fi
COPY --from=sliamb/paopao-pref /usr/bin/paopao-pref /usr/bin/
WORKDIR /data
RUN paopao-pref -inrule /src/inrule.txt -outrule /data/inrule.txt
RUN cat /data/inrule.txt >> /src/domains.txt
RUN paopao-pref -inrule /src/domains.txt -outrule /data/domains.txt

FROM alpine:edge
RUN apk add --no-cache curl xz
COPY --from=sliamb/paopao-pref /usr/bin/paopao-pref /usr/bin/
COPY --from=sliamb/prebuild-paopaodns /src/mosdns /usr/bin/
COPY --from=download /src/Country-only-cn-private.mmdb /data/
COPY --from=download /data/inrule.txt /data/
COPY --from=download /data/domains.txt /data/
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
    SYSDNS="no" \
    FILE_OUTPUT=yes
WORKDIR /data/
CMD sh mark_update.sh