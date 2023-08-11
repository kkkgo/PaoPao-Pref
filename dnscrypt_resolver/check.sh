#!/bin/sh
sudo apt-get update
sudo apt-get -qq install dnscrypt-proxy git
git clone https://github.com/DNSCrypt/dnscrypt-resolvers.git --depth 1 /dnscrypt-resolvers

