#!/bin/sh
sudo apt-get update
sudo apt-get -qq install dnscrypt-proxy git dnsutils
git clone https://github.com/DNSCrypt/dnscrypt-resolvers.git --depth 1 /tmp/dnscrypt-resolvers
grep -E "##" /tmp/dnscrypt-resolvers/v3/public-resolvers.md >/tmp/dnscrypt-resolvers/dnstest_alldns.txt
cut -d" " -f2 /tmp/dnscrypt-resolvers/dnstest_alldns.txt >/tmp/name_list.txt
cat /tmp/name_list.txt

# config dnscrypt
#gen dns toml
git clone https://github.com//DNSCrypt/dnscrypt-proxy --depth 1 /tmp/dnscrypt-proxy
grep -v "#" /dnscrypt-proxy/dnscrypt-proxy/example-dnscrypt-proxy.toml | grep . >/tmp/dnscrypt-proxy/dnsex.toml
sed -i -r "s/listen_addresses.+/listen_addresses = ['0.0.0.0:5302']/g" /tmp/dnscrypt-proxy/dnsex.toml
sed -i -r "s/server_names.+//g" /tmp/dnscrypt-proxy/dnsex.toml

cat  /tmp/dnscrypt-proxy/dnsex.toml