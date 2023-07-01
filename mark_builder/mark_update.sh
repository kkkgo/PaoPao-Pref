#!/bin/sh
IPREX4='([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])'

gen_dns() {
    if [ "$SYSDNS" = "yes" ]; then
        cat /etc/resolv.conf | grep -Eo "$IPREX4" >/tmp/dns_list.txt
    else
        touch /tmp/delay.txt
        while read dnsserver; do
            echo "Test "$dnsserver
            sed "s/{ser1}/$dnsserver/g" test_cn.yaml | sed "s/#dns_check//g" >/tmp/test_cn.yaml
            mosdns start -d /tmp -c test_cn.yaml >/dev/null 2>&1 &
            sleep 1
            delay=$(paopao-pref -server 127.0.0.1 -port 5304 -delay) && echo "$delay"",""$dnsserver" >>/tmp/delay.txt && echo "$dnsserver"": ""$delay"" ms"
            killall mosdns
        done <dns_list.txt
        cat /tmp/delay.txt
        sort -n /tmp/delay.txt | cut -d "," -f2 | head -3 >/tmp/dns_list.txt
    fi
    touch /tmp/force_list.txt
    while read dnsserver; do
        sed "s/{ser1}/$dnsserver/g" test_cn.yaml | sed "s/#dns_check//g" >/tmp/test_cn.yaml
        mosdns start -d /tmp -c test_cn.yaml >/dev/null 2>&1 &
        sleep 1
        paopao-pref -server 127.0.0.1 -port 5304 -delay -v
        killall mosdns
    done </tmp/dns_list.txt
    ser_num=$(cat /tmp/dns_list.txt | wc -l)
    if [ "$ser_num" = "0" ]; then
        echo "no dns available."
        cat /tmp/dns_list.txt
        exit
    fi
    ser1=$(head -1 /tmp/dns_list.txt)
    ser2=$(head -2 /tmp/dns_list.txt | tail -1)
    ser3=$(tail -1 /tmp/dns_list.txt)
    cp test_cn.yaml /tmp/gen.yaml
    if [ "$ser_num" -gt 0 ]; then
        sed -i "s/{ser1}/$ser1/g" /tmp/gen.yaml
    fi

    if [ "$ser_num" -gt 1 ]; then
        sed -i "s/#ser_num2//g" /tmp/gen.yaml
        sed -i "s/{ser2}/$ser2/g" /tmp/gen.yaml
    fi

    if [ "$ser_num" -gt 2 ]; then
        sed -i "s/#ser_num3//g" /tmp/gen.yaml
        sed -i "s/{ser3}/$ser3/g" /tmp/gen.yaml
    fi
}

gen_nocn_list() {
    if [ ! -d "/predata/" ]; then
        cat /data/fwrule.txt >/tmp/force_list.txt
    else
        cat /predata/force_list_global.txt >/tmp/force_list.txt
    fi
}

gen_cn_list() {
    if [ ! -d "/predata/" ]; then
        echo "" >/tmp/force_list.txt
    else
        cat /predata/force_list_cn.txt >/tmp/force_list.txt
    fi
}

mosdns_gen_mark_start() {
    sed "s/#gen_mark//g" /tmp/gen.yaml >/tmp/gen_mark.yaml
    mosdns start -d /tmp -c gen_mark.yaml &
    sleep 1
    ps
}

mosdns_gen_cn_start() {
    killall mosdns
    sleep 1
    sed "s/#icp_mark//g" /tmp/gen.yaml >/tmp/icp_mark.yaml
    mosdns start -d /tmp -c icp_mark.yaml &
    sleep 1
    ps
}

gen_pfdata_mark() {
    if [ ! -d "/predata/" ]; then
        cat /data/fwrule.txt >/tmp/pfdata.txt
        echo "" >>/tmp/pfdata.txt
        echo "" >>/tmp/pfdata.txt
        cat /data/domains.txt >>/tmp/pfdata.txt
        echo "" >>/tmp/pfdata.txt
        paopao-pref -inrule /tmp/pfdata.txt -outrule /data/pfdata.txt
    else
        cat /predata/pfdata.txt >/data/pfdata.txt
    fi
}

pref_start_mark() {
    echo "Start pref mark..."
    if [ -f domains_ok.txt ]; then
        rm domains_ok.txt
        touch domains_ok.txt
    fi
    paopao-pref -file /data/pfdata.txt -server 127.0.0.1 -port 5304 -v >/tmp/pref.log
}

pref_start_cn() {
    echo "Start pref cn..."
    if [ -f domains_ok.txt ]; then
        rm domains_ok.txt
        touch domains_ok.txt
    fi
    paopao-pref -file /data/domains_raw.txt -server 127.0.0.1 -port 5304 -v >/tmp/pref.log
}

global_debug() {
    mkdir -p /pub/debug/global
    cp /tmp/pref.log /pub/debug/global/
    cp /data/domains_ok.txt /pub/debug/global/
    cp /data/pfdata.txt /pub/debug/global/
    cp /data/domains_raw.txt /pub/debug/
}

global_cn() {
    mkdir -p /pub/debug/cn
    cp /tmp/pref.log /pub/debug/cn/
    cp /data/domains_ok.txt /pub/debug/cn/
}

gen_dat() {
    if [ ! -d "/predata/" ]; then
        cat /data/fwrule.txt >/tmp/global_mark.txt
        echo "" >>/tmp/global_mark.txt
        echo "" >>/tmp/global_mark.txt
        cat /data/domains_ok.txt >>/tmp/global_mark.txt
        echo "" >>/tmp/global_mark.txt
        paopao-pref -inrule /tmp/global_mark.txt -outrule /data/global_mark.dat
    else
        cat /predata/force_list_global.txt >/tmp/global_mark.txt
        echo "" >>/tmp/global_mark.txt
        cat /data/domains_ok.txt >>/tmp/global_mark.txt
        echo "" >>/tmp/global_mark.txt
        paopao-pref -inrule /tmp/global_mark.txt -outrule /data/global_mark_global.dat
        if [ "$TEST" = "debug" ]; then
            paopao-pref -an -inrule /data/global_mark_global.dat -outrule /pub/debug/global/global_mark_global_analyze.txt
        fi
        gen_cn_list
        mosdns_gen_cn_start
        pref_start_cn
        if [ "$TEST" = "debug" ]; then
            global_cn
        fi
    fi

    cat /predata/cn.txt >/tmp/global_mark_cn.txt
    echo "" >>/tmp/global_mark_cn.txt
    cat /data/domains_ok.txt >>/tmp/global_mark_cn.txt
    echo "" >>/tmp/global_mark_cn.txt
    paopao-pref -inrule /tmp/global_mark_cn.txt -outrule /data/global_mark_cn.dat

    if [ "$TEST" = "debug" ]; then
        paopao-pref -an -inrule /data/global_mark_cn.dat -outrule /pub/debug/cn/global_mark_cn_analyze.txt
    fi
    sed -ir "s/^domain:/##@@domain:/g" /data/global_mark_cn.dat
    echo "" >/tmp/global_mark.dat
    cat /data/global_mark_global.dat >>/tmp/global_mark.dat
    echo "" >>/tmp/global_mark.dat
    cat /data/global_mark_cn.dat >>/tmp/global_mark.dat
    echo "" >>/tmp/global_mark.dat
    cp /tmp/global_mark.dat /data/global_mark.dat
}

hash_dat() {
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
gen_nocn_list
mosdns_gen_mark_start
gen_pfdata_mark
pref_start_mark
if [ "$TEST" = "debug" ]; then
    global_debug
fi

gen_dat
hash_dat
