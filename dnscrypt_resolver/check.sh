#!/bin/sh
sudo apt-get update
sudo apt-get -qq install dnscrypt-proxy git
git clone https://github.com/DNSCrypt/dnscrypt-resolvers.git --depth 1 /tmp/dnscrypt-resolvers
grep -E "##" /tmp/dnscrypt-resolvers/v3/public-resolvers.md >/tmp/dnscrypt-resolvers/dnstest_alldns.txt
cat /tmp/dnscrypt-resolvers/dnstest_alldns.txt
