#!/bin/sh
mkdir -p /pub
IPREX4='([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])'

# gen test_cn for /tmp/gen.yaml
gen_dns() {
    cat /data/dns_list.txt /etc/resolv.conf | grep -Eo "$IPREX4" >>/tmp/dns_list.txt
    touch /tmp/delay.txt
    while read dnsserver; do
        echo "Test "$dnsserver
        sed "s/1.2.3.4/$dnsserver/g" test_cn.yaml >/tmp/test_cn.yaml
        mosdns start -d /tmp -c test_cn.yaml >/dev/null 2>&1 &
        sleep 1
        delay=$(paopao-pref -server 127.0.0.1 -port 5301 -delay) && echo "$delay"",""$dnsserver" >>/tmp/delay.txt && echo "$dnsserver"": ""$delay"" ms"
        killall mosdns
    done </tmp/dns_list.txt
    cat /tmp/delay.txt
    sort -n /tmp/delay.txt | cut -d "," -f2 | head -3 >/tmp/dns_list_gen.txt
    while read dnsserver; do
        sed "s/1\.2\.3\.4/$dnsserver/g" test_cn.yaml >/tmp/test_cn.yaml
        mosdns start -d /tmp -c test_cn.yaml >/dev/null 2>&1 &
        sleep 1
        paopao-pref -server 127.0.0.1 -port 5301 -delay -v
        killall mosdns
    done </tmp/dns_list_gen.txt
    ser_num=$(cat /tmp/dns_list_gen.txt | wc -l)
    if [ "$ser_num" = "0" ]; then
        echo "no dns available."
        cat /tmp/dns_list_gen.txt
        exit
    fi
    ser1=$(head -1 /tmp/dns_list_gen.txt)
    ser2=$(head -2 /tmp/dns_list_gen.txt | tail -1)
    cp test_cn.yaml /tmp/gen.yaml
    sed -i "s/6\.7\.8\.9/$ser1/g" /tmp/gen.yaml
    if [ "$ser_num" -gt 1 ]; then
        sed -i "s/9\.8\.7\.6/$ser2/g" /tmp/gen.yaml
    else
        sed -i "/9\.8\.7\.6/d" /tmp/gen.yaml
    fi
}

mosdns_gen_mark_start() {
    killall mosdns
    sed "s/#gen_mark//g" /tmp/gen.yaml >/tmp/gen_mark.yaml
    mosdns start -d /tmp -c gen_mark.yaml &
    sleep 1
    ps
}

pref_start_mark() {
    echo "Start pref mark..."
    if [ -f domains_ok.txt ]; then
        rm domains_ok.txt
        touch domains_ok.txt
    fi
    paopao-pref -file /data/topdomains.rules -server 127.0.0.1 -port 5302
}

pref_start_cn() {
    echo "Start pref cn..."
    if [ -f domains_ok.txt ]; then
        rm domains_ok.txt
        touch domains_ok.txt
    fi
    paopao-pref -file /data/topdomains.txt -server 127.0.0.1 -port 5303
}

gen_global() {
    gen_global_txt="/tmp/global.data.txt"
    cat /data/global.rules >$gen_global_txt
    echo "" >>$gen_global_txt
    cat /data/domains_ok.txt >>$gen_global_txt
    echo "" >>$gen_global_txt
    paopao-pref -inrule $gen_global_txt -outrule /data/global.data
    if [ "$TEST" = "debug" ]; then
        mkdir -p /pub/debug/global/
        touch /pub/debug/global/global.data.txt
        paopao-pref -an -inrule /data/global.data -outrule /pub/debug/global/global.data.txt
    fi
}
gen_cn() {
    gen_cn_txt="/tmp/cn.data.txt"
    cat /data/global.cnfilter.rules >$gen_cn_txt
    echo "" >>$gen_cn_txt
    cat /data/cn.rules >>$gen_cn_txt
    echo "" >>$gen_cn_txt
    cat /data/domains_ok.txt >>$gen_cn_txt
    echo "" >>$gen_cn_txt
    paopao-pref -inrule $gen_cn_txt -outrule /data/cn.data
    if [ "$TEST" = "debug" ]; then
        mkdir -p /pub/debug/cn/
        touch /pub/debug/cn/cn.data.txt
        paopao-pref -an -inrule /data/cn.data -outrule /pub/debug/cn/cn.data.txt
    fi
}

hash_dat() {
    paopao-pref -gbfile /data/global.data -grfile /data/global.rules -crfile /data/cn.rules -cnfile /data/cn.data -comp /data/global_mark.dat.tmp
    cat /data/force_cn.rules /data/global_mark.dat.tmp | sort -u >/data/global_mark.dat
    cd /data || exit
    if [ "$TEST" = "debug" ]; then
        cp global_mark.dat /pub/global_mark_raw.dat
    fi
    xz -9 -e global_mark.dat
    datsha=$(sha512sum global_mark.dat.xz | cut -d" " -f1)
    echo -n $datsha >sha.txt
    shasize=$(wc -c <sha.txt)
    dd if=/dev/zero of=sha.txt bs=1 count=$((1024 - shasize)) seek=$shasize conv=notrunc
    cat global_mark.dat.xz sha.txt >global_mark.dat
    sha256sum global_mark.dat | cut -d" " -f1 >/pub/global_mark.dat.sha256sum
    mv global_mark.dat /pub
}

gen_dns
mosdns_gen_mark_start
pref_start_mark
gen_global
pref_start_cn
gen_cn
hash_dat
