#!/bin/sh
if [ -f /predata/global.hook.rules ]; then
    cat /predata/global.hook.rules >/data/global.hook.raw
fi
if [ -f /predata/cn.hook.rules ]; then
    cat /predata/cn.hook.rules >/data/cn.hook.raw
fi
mkdir -p /data
curl -sLo /data/Country-only-cn-private.mmdb https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb &&
    mmdb_hash=$(sha256sum /data/Country-only-cn-private.mmdb | grep -Eo "[a-zA-Z0-9]{64}" | head -1) &&
    mmdb_down_hash=$(curl -s https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb.sha256sum | grep -Eo "[a-zA-Z0-9]{64}" | head -1) &&
    if [ "$mmdb_down_hash" != "$mmdb_hash" ]; then
        cp /mmdb_down_hash_error .
        exit
    fi

curl -sLo /data/inrule_base64.txt https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt &&
    domains_size=$(wc -c <"/data/inrule_base64.txt") &&
    if [ "$domains_size" -gt 100000 ]; then echo "domains_size pass."; else
        echo "domains_size failed"
        cp /domains_size /
        exit
    fi &&
    base64 -d /data/inrule_base64.txt >/data/inrule.txt

curl -sLo /data/topdomains.data https://github.com/kkkgo/PaoPao-Pref/raw/main/domains.txt &&
    domains_size=$(wc -c <"/data/topdomains.data") &&
    if [ "$domains_size" -gt 10000000 ]; then echo "domains_size pass."; else
        echo "domains_size failed"
        cp /domains_size /
        exit
    fi

curl -sLo /data/cn.txt https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/direct-list.txt &&
    domains_size=$(wc -c <"/data/cn.txt") &&
    if [ "$domains_size" -gt 50000 ]; then echo "domains_size pass."; else
        echo "domains_size failed"
        cp /domains_size /
        exit
    fi
curl -sLo /data/applecn.txt https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/apple-cn.txt &&
    domains_size=$(wc -c <"/data/applecn.txt") &&
    if [ "$domains_size" -gt 50 ]; then echo "domains_size pass."; else
        echo "domains_size failed"
        cp /domains_size /
        exit
    fi
echo "" >>/data/cn.rules.txt
grep -Ev "^regexp:" /data/applecn.txt >>/data/cn.rules.txt

curl -sLo /data/proxy.txt https://raw.githubusercontent.com/Loyalsoldier/v2ray-rules-dat/release/proxy-list.txt &&
    domains_size=$(wc -c <"/data/proxy.txt") &&
    if [ "$domains_size" -gt 10000 ]; then echo "domains_size pass."; else
        echo "domains_size failed"
        cp /domains_size /
        exit
    fi
grep -Ev "^regexp:" /data/proxy.txt | grep "." >/data/proxy.rules.txt

paopao-pref -inrule /data/cn.rules.txt -outrule /data/cn.rules

# global.rules: hook+fw+proxy
touch /data/global.hook.raw
echo "" >>/data/global.hook.raw
cat /data/inrule.txt >>/data/global.hook.raw
echo "" >>/data/global.hook.raw
cat /data/proxy.rules.txt >>/data/global.hook.raw
echo "" >>/data/global.hook.raw
paopao-pref -inrule /data/global.hook.raw -outrule /data/global.rules

touch /data/cn.hook.raw
echo "" >>/data/cn.hook.raw
cat /data/cn.rules >>/data/cn.hook.raw
echo "" >>/data/cn.hook.raw
paopao-pref -inrule /data/cn.hook.raw -outrule /data/global.cnfilter.rules

touch /data/skip.raw
echo "" >>/data/skip.raw
cat /data/global.rules >>/data/skip.raw
echo "" >>/data/skip.raw
cat /data/global.cnfilter.rules >>/data/skip.raw
echo "" >>/data/skip.raw
paopao-pref -inrule /data/skip.raw -outrule /data/mark.skip.rules

# slim know top
cp /data/topdomains.data /data/topdomains.txt
echo "" >>/data/topdomains.data
cat /data/mark.skip.rules >>/data/topdomains.data
echo "" >>/data/topdomains.data
paopao-pref -inrule /data/topdomains.data -outrule /data/topdomains.rules
