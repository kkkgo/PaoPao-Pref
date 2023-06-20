#!/bin/sh
IPREX4='([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.([0-9]{1,2}|1[0-9][0-9]|2[0-4][0-9]|25[0-5])'
touch /tmp/inrule.txt
touch /data/inrule.txt
mkdir -p /predata
touch /predata/force_nocn_list.txt
if [ "$EXT" = "yes" ]; then
    if [ -f /pub/tlds.txt ]; then
        echo apply tlds.
        cat /pub/tlds.txt >>/data/inrule.txt
        echo "" >>/data/inrule.txt
        echo "" >>/data/inrule.txt
    fi
    if [ -f /pub/cn.txt ]; then
        echo apply cn.
        cat /pub/cn.txt >>/predata/force_nocn_list.txt
        echo "" >>/predata/force_nocn_list.txt
        echo "" >>/predata/force_nocn_list.txt
    fi
else
    cat /data/inrule.txt >>/predata/force_nocn_list.txt
    echo "" >>/predata/force_nocn_list.txt
    echo "" >>/predata/force_nocn_list.txt
    cat /data/inrule.txt >/predata/inrule.txt
fi
paopao-pref -inrule /predata/force_nocn_list.txt -outrule /tmp/force_nocn_list.txt
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
sed "s/#gen_mark//g" /tmp/gen.yaml >/tmp/gen_mark.yaml
mosdns start -d /tmp -c gen_mark.yaml &
sleep 1

ps
cat /tmp/force_nocn_list.txt >>/domains.txt
paopao-pref -inrule /domains.txt -outrule /data/domains.txt

echo "Start pref..."
if [ -f domains_ok.txt ]; then
    rm domains_ok.txt
    touch domains_ok.txt
fi
paopao-pref -server 127.0.0.1 -port 5304 -v >/tmp/pref.log
mkdir -p /pub
if [ "$TEST" = "debug" ]; then
    cp /tmp/pref.log /pub/
    cp /data/domains_ok.txt /pub/
    cp /data/domains.txt /pub/
    cp /domains.txt /pub/domains_raw.txt
fi
cat /predata/inrule.txt >>/data/domains_ok.txt
paopao-pref -inrule /data/domains_ok.txt -outrule /data/global_mark.dat
killall mosdns
if [ "$TEST" = "debug" ]; then
    cp /data/global_mark.dat /pub/raw.dat
    paopao-pref -an -inrule /data/global_mark.dat -outrule /pub/global_mark_analyze.txt
    cut -d":" -f1 /pub/global_mark_analyze.txt >/pub/global_mark_analyze_raw.txt
    sed 'p; s/.*/www.&/' /pub/global_mark_analyze_raw.txt >/pub/global_mark_analyze_icptest.txt
    sed "s/#icp_mark//g" /tmp/gen.yaml >/tmp/icp_mark.yaml
    mosdns start -d /tmp -c icp_mark.yaml &
    sleep 1
    if [ -f domains_ok.txt ]; then
        rm domains_ok.txt
        touch domains_ok.txt
    fi
    cat /pub/global_mark_analyze_icptest.txt >/data/domains.txt
    paopao-pref -server 127.0.0.1 -port 5304 -v >/tmp/pref_icp.log
    paopao-pref -an -inrule /data/domains_ok.txt -outrule /pub/domains_ok_icp.txt
    cp /tmp/pref_icp.log /pub/
fi

xz -9 -e global_mark.dat
datsha=$(sha512sum global_mark.dat.xz | cut -d" " -f1)
echo -n $datsha >sha.txt
shasize=$(wc -c <sha.txt)
dd if=/dev/zero of=sha.txt bs=1 count=$((1024 - shasize)) seek=$shasize conv=notrunc
cat global_mark.dat.xz sha.txt >global_mark.dat
sha256sum global_mark.dat | cut -d" " -f1 >/pub/global_mark.dat.sha256sum
mv global_mark.dat /pub
