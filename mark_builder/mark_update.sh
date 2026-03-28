#!/bin/sh
mkdir -p /pub
IPREX4='([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])'

log() { echo "[$(date '+%H:%M:%S')] $*"; }

# Test DNS servers directly with paopao-pref CN check mode
gen_dns() {
    cat /data/dns_list.txt /etc/resolv.conf | grep -Eo "$IPREX4" >>/tmp/dns_list.txt
    touch /tmp/delay.txt
    while read -r dnsserver; do
        log "Testing $dnsserver"
        delay=$(paopao-pref -server "$dnsserver" -port 53 \
            -cndat /data/CN-local.dat -cnmode check -delay) && \
            echo "${delay},${dnsserver}" >>/tmp/delay.txt && \
            log "$dnsserver: ${delay} ms"
    done </tmp/dns_list.txt
    cat /tmp/delay.txt
    sort -n /tmp/delay.txt | cut -d "," -f2 | head -3 >/tmp/dns_list_gen.txt
    # Verify selected servers
    while read -r dnsserver; do
        paopao-pref -server "$dnsserver" -port 53 \
            -cndat /data/CN-local.dat -cnmode check -delay -v
    done </tmp/dns_list_gen.txt
    ser_num=$(wc -l </tmp/dns_list_gen.txt)
    if [ "$ser_num" = "0" ]; then
        log "no dns available."
        cat /tmp/dns_list_gen.txt
        exit
    fi
    ser1=$(head -1 /tmp/dns_list_gen.txt)
    log "Selected DNS server: $ser1"
}

# Gen global_mark: find domains that resolve to non-CN/non-PRIVATE IPs
pref_start_mark() {
    log "Start pref mark..."
    if [ -f domains_ok.txt ]; then
        rm domains_ok.txt
        touch domains_ok.txt
    fi
    paopao-pref -file /data/topdomains.rules -server "$ser1" -port 53 \
        -cndat /data/CN-local.dat -cnmode mark \
        -skipfile /data/alreadymark.skip.rules
}

# Gen /data/global.data: /data/global.rules + pref_start_mark
gen_global() {
    gen_global_txt="/tmp/global.data.txt"
    {
        cat /data/global.rules
        echo ""
        cat /data/domains_ok.txt
        echo ""
    } >"$gen_global_txt"
    paopao-pref -inrule $gen_global_txt -outrule /data/global.data
    if [ "$TEST" = "debug" ]; then
        mkdir -p /pub/debug/global/
        touch /pub/debug/global/global.data.txt
        paopao-pref -an -inrule /data/global.data -outrule /pub/debug/global/global.data.txt
    fi
}

# Gen global_mark_cn: find domains that resolve to CN IPs
pref_start_cn() {
    log "Start pref cn..."
    if [ -f domains_ok.txt ]; then
        rm domains_ok.txt
        touch domains_ok.txt
    fi
    paopao-pref -file /data/topdomains.txt -server "$ser1" -port 53 \
        -cndat /data/CN-local.dat -cnmode cnmark \
        -skipfile /data/alreadymark.skip.rules
}

# Gen /data/cn.data: /data/cn.rules(direct + cn_mark) + global.cnfilter.rules(cn.hook.rules + cn.rules) + topdomains(cn)
gen_cn() {
    gen_cn_txt="/tmp/cn.data.txt"
    {
        cat /data/global.cnfilter.rules
        echo ""
        cat /data/cn.rules
        echo ""
        cat /data/domains_ok.txt
        echo ""
    } >"$gen_cn_txt"
    paopao-pref -inrule $gen_cn_txt -outrule /data/cn.data
    if [ "$TEST" = "debug" ]; then
        mkdir -p /pub/debug/cn/
        touch /pub/debug/cn/cn.data.txt
        paopao-pref -an -inrule /data/cn.data -outrule /pub/debug/cn/cn.data.txt
    fi
}

hash_dat() {
    paopao-pref -gbfile /data/global.data -grfile /data/global.rules -crfile /data/cn.rules -cnfile /data/cn.data -comp /data/global_mark.dat
    cd /data || exit
    if [ "$TEST" = "debug" ]; then
        cp global_mark.dat /pub/global_mark_raw.dat
    fi
    xz -9 -e global_mark.dat
    datsha=$(sha512sum global_mark.dat.xz | cut -d" " -f1)
    printf '%s' "$datsha" >sha.txt
    shasize=$(wc -c <sha.txt)
    dd if=/dev/zero of=sha.txt bs=1 count=$((1024 - shasize)) seek=$shasize conv=notrunc
    cat global_mark.dat.xz sha.txt >global_mark.dat
    sha256sum global_mark.dat | cut -d" " -f1 >/pub/global_mark.dat.sha256sum
    mv global_mark.dat /pub
}

gen_dns
pref_start_mark
gen_global
pref_start_cn
gen_cn
hash_dat
