#!/bin/sh

mkdir -p /data

# Get the latest mmdb: /data/Country-only-cn-private.mmdb
git clone --depth 1 -b main https://github.com/kkkgo/Country-only-cn-private.mmdb /data/mmdb_repo &&
    cp /data/mmdb_repo/Country-only-cn-private.mmdb /data/Country-only-cn-private.mmdb &&
    mmdb_hash=$(sha256sum /data/Country-only-cn-private.mmdb | grep -Eo "[a-zA-Z0-9]{64}" | head -1) &&
    mmdb_down_hash=$(grep -Eo "[a-zA-Z0-9]{64}" /data/mmdb_repo/Country-only-cn-private.mmdb.sha256sum | head -1) &&
    rm -rf /data/mmdb_repo &&
    if [ "$mmdb_down_hash" = "$mmdb_hash" ]; then echo "mmdb_size pass."; else
        cp /mmdb_down_hash_error .
        exit
    fi

# Get the latest topdomains: /data/topdomains.data
curl -sLo /data/top-1m.csv.zip http://s3-us-west-1.amazonaws.com/umbrella-static/top-1m.csv.zip &&
    unzip top-1m.csv.zip &&
    cut -d"," -f2 top-1m.csv >/data/topdomains.data &&
    domains_size=$(wc -c <"/data/topdomains.data") &&
    if [ "$domains_size" -gt 20000000 ]; then echo "domains_size pass."; else
        echo "domains_size failed: topdomains.data"
        cp /domains_size /domains_size_failed_topdomains.data
        exit
    fi

# Get the latest proxy.rules.: /data/global.nofilter.rules
git clone --depth 1 -b master https://github.com/gfwlist/gfwlist /data/gfwlist_repo &&
    cp /data/gfwlist_repo/list.txt /data/proxy.rules.txt &&
    rm -rf /data/gfwlist_repo &&
    domains_size=$(wc -l <"/data/proxy.rules.txt") &&
    if [ "$domains_size" -gt 4000 ]; then echo "domains_size pass."; else
        echo "domains_size failed: proxy.rules.txt"
        cp /domains_size /domains_size_failed_proxy.rules.txt
        exit
    fi
echo "" >>/data/proxy.rules.txt
if [ -f /predata/global.hook.rules ]; then
    cat /predata/global.hook.rules >>/data/proxy.rules.txt
fi
paopao-pref -inrule /data/proxy.rules.txt -outrule /data/global.nofilter.rules

# Get the latest /data/cn.rules: direct + cn_mark
git clone --depth 1 -b release https://github.com/Loyalsoldier/v2ray-rules-dat /data/v2ray_repo &&
    cp /data/v2ray_repo/direct-list.txt /data/cn.txt &&
    rm -rf /data/v2ray_repo &&
    domains_size=$(wc -c <"/data/cn.txt") &&
    if [ "$domains_size" -gt 50000 ]; then echo "domains_size pass."; else
        echo "domains_size failed: cn.txt"
        cp /domains_size /domains_size_failed_cn.txt
        exit
    fi
if [ -f /predata/cn_mark.rules ]; then
    echo "" >>/data/cn.txt
    cat /predata/cn_mark.rules >>/data/cn.txt
fi
paopao-pref -inrule /data/cn.txt -outrule /data/cn.rules


# Gen /data/global.cnfilter.rules : cn.hook.rules + cn.rules
if [ -f /predata/cn.hook.rules ]; then
    cat /predata/cn.hook.rules >/data/cn.hook.raw
fi
touch /data/cn.hook.raw
echo "" >>/data/cn.hook.raw
cat /data/cn.rules >>/data/cn.hook.raw
echo "" >>/data/cn.hook.raw
paopao-pref -inrule /data/cn.hook.raw -outrule /data/global.cnfilter.rules

# Gen /data/global.rules : global.nofilter.rules - global.cnfilter.rules
paopao-pref -inrule /data/global.nofilter.rules -filter /data/global.cnfilter.rules -outrule /data/global.rules

# Gen alreadymark.skip.rules : global.rules + global.cnfilter.rules
touch /data/skip.raw
echo "" >>/data/skip.raw
cat /data/global.rules >>/data/skip.raw
echo "" >>/data/skip.raw
cat /data/global.cnfilter.rules >>/data/skip.raw
echo "" >>/data/skip.raw
paopao-pref -inrule /data/skip.raw -outrule /data/alreadymark.skip.rules

# cp topdomains.data to topdomains.txt 
cp /data/topdomains.data /data/topdomains.txt

# Gen topdomains.rules slim with alreadymark.skip.rules : topdomains.data + alreadymark.skip.rules
echo "" >>/data/topdomains.data
cat /data/alreadymark.skip.rules >>/data/topdomains.data
echo "" >>/data/topdomains.data
paopao-pref -inrule /data/topdomains.data -outrule /data/topdomains.rules
