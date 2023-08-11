#!/bin/sh
git clone https://github.com/DNSCrypt/dnscrypt-resolvers.git --depth 1 /tmp/dnscrypt-resolvers
grep -E "##" /tmp/dnscrypt-resolvers/v3/public-resolvers.md >/tmp/dnscrypt-resolvers/dnstest_alldns.txt
cut -d" " -f2 /tmp/dnscrypt-resolvers/dnstest_alldns.txt | sort -u >/tmp/name_list.txt
echo "" >>/tmp/name_list.txt
cat /tmp/name_list.txt

# config dnscrypt
#gen dns toml
git clone https://github.com//DNSCrypt/dnscrypt-proxy --depth 1 /tmp/dnscrypt-proxy
grep -v "#" /tmp/dnscrypt-proxy/dnscrypt-proxy/example-dnscrypt-proxy.toml | grep . >/tmp/dnsex.toml
sed -i -r "s/listen_addresses.+/listen_addresses = ['0.0.0.0:5302']/g" /tmp/dnsex.toml
sed -i -r "s/^server_names.+//g" /tmp/dnsex.toml
cat /tmp/dnsex.toml
dnscrypt-proxy -config /tmp/dnsex.toml &
sleep 5

local_lookup() {
    killall dnscrypt-proxy >/dev/null 2>&1 &
    server_name=$1
    domain_name=$2
    sed "1i server_names = [ '$server_name' ]" /tmp/dnsex.toml >/tmp/test_now.toml
    dnscrypt-proxy -config /tmp/test_now.toml >/dev/null 2>&1 &
    test_res=$(dig @127.0.0.1 -p5302 "$domain_name")
    killall dnscrypt-proxy >/dev/null 2>&1 &
    echo "$test_res"
}

# test
touch /tmp/bad_new.txt
cat /data/ban_list.txt >>/tmp/bad_new.txt
testrec=$(nslookup local.03k.org)
if echo "$testrec" | grep -q "10.9.8.7"; then
    echo "Ready to test..."
    while read sdns; do
        test=$(local_lookup "$sdns" local.03k.org)
        if echo "$test" | grep -q "10.9.8.7"; then
            echo "$sdns"": OK."
        else
            again_test=$(local_lookup "$sdns" gmail.com)
            if echo "$again_test" | grep -q "smtp"; then
                echo "$sdns"": LOCAL BAD."
                echo "$sdns" >>/tmp/bad_new.txt
            else
                echo "$sdns"": CONNECT BAD."
            fi
        fi
    done </tmp/name_list.txt
else
    echo "Test record failed.""$testrec"
fi
sort -u /tmp/bad_new.txt | grep -E "[a-z]" >/data/ban_list.txt
