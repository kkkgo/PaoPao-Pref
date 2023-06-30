#!/bin/sh
mkdir -p /src
curl -sLo /src/Country-only-cn-private.mmdb https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb &&
    mmdb_hash=$(sha256sum /src/Country-only-cn-private.mmdb | grep -Eo "[a-zA-Z0-9]{64}" | head -1) &&
    mmdb_down_hash=$(curl -s https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb.sha256sum | grep -Eo "[a-zA-Z0-9]{64}" | head -1) &&
    if [ "$mmdb_down_hash" != "$mmdb_hash" ]; then
        cp /mmdb_down_hash_error .
        exit
    fi
curl -sLo /src/inrule_base64.txt https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt &&
    domains_size=$(wc -c <"/src/inrule_base64.txt") &&
    if [ "$domains_size" -gt 100000 ]; then echo "domains_size pass."; else
        echo "domains_size failed"
        cp /domains_size /
        exit
    fi &&
    base64 -d /src/inrule_base64.txt >/src/inrule.txt
curl -sLo /src/domains.txt https://github.com/kkkgo/PaoPao-Pref/raw/main/domains.txt &&
    domains_size=$(wc -c <"/src/domains.txt") &&
    if [ "$domains_size" -gt 10000000 ]; then echo "domains_size pass."; else
        echo "domains_size failed"
        cp /domains_size /
        exit
    fi
mkdir -p /data

paopao-pref -inrule /src/inrule.txt -outrule /data/fwrule.txt
paopao-pref -inrule /src/domains.txt -outrule /data/domains.txt
cp /src/domains.txt /data/domains_raw.txt