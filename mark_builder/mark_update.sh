#!/bin/sh
cp /pub/domains.txt .
curl -sLo /tmp/Country-only-cn-private.mmdb https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb
mmdb_hash=$(sha256sum /tmp/Country-only-cn-private.mmdb | grep -Eo "[a-zA-Z0-9]{64}" | head -1)
mmdb_down_hash=$(curl -s https://raw.githubusercontent.com/kkkgo/Country-only-cn-private.mmdb/main/Country-only-cn-private.mmdb.sha256sum | grep -Eo "[a-zA-Z0-9]{64}" | head -1)
if [ "$mmdb_down_hash" != "$mmdb_hash" ]; then
    cp /mmdb_down_hash_error .
    exit
fi
curl -sLo /tmp/inrule_base64.txt https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt
domains_size=$(wc -c <"/tmp/inrule_base64.txt")
if [ "$domains_size" -gt 100000 ]; then
    echo "domains_size pass."
else
    echo "domains_size failed"
    cp /domains_size /
    exit
fi
base64 -d /tmp/inrule_base64.txt > /tmp/inrule.txt 
tld_list="
||.jp
||.tw
||.hk
||.de
||.us
||.ca
||.uk
||.au
||.ru
||.google
||.in
||.gov
||.eu
||.mil
||.fr
||.edu
||.nl
||.se
||.kr
||.ch
||.be
||.sg
||.nz
||.es
||.xxx
||.ws
||.dk
||.at
||.vn
||.pl
||.br
||.mx
||.ir
||.tr
||.cz
||.ms
||.th
||.gr
||.ro
||.il
||.hu
||.cl
||.ie
||.ph
||.sk
||.fi
||.ua
||.za
||.goog
||.cr
||.pe
||.bg
||.lt
||.rs
||.ae
||.hr
||.ge
||.sa
||.digital
||.pk
||.is
||.lu
||.by
||.sh
||.kz
||.ee
||.la
||.ec
||.systems
||.si
||.tools
||.do
||.support
||.lk
||.st
||.click
||.lv
||.gt
||.ng
||.ve
||.aero
||.solutions
||.page
||.li
||.bank
||.md
||.nu
||.su
||.ke
||.stream
||.jobs
||.bz
||.travel
||.rocks
||.bid
||.am
||.uz
||.software
||.ag
||.sap
||.hn
||.events
||.pics
||.eg
||.lol
||.re
||.cx
||.uy
||.ba
||.buzz
||.agency
||.pa
||.az
||.qa
||.ac
||.health
||.al
||.ma
||.sc
||.wtf
||.vc
||.tt
||.build
||.bd
||.land
||.tk
||.help
||.gl
||.tc
||.sv
||.kg
||.ovh
||.cm
||.nyc
||.mm
||.gs
||.moe
||.tm
||.om
||.cy
||.as
||.gh
||.bm
||.vg
||.sky
||.farm
||.abbott
||.town
||.lb
||.bot
||.sx
||.pm
||.ni
||.money
||.london
||.gq
||.cfd
||.tl
||.sbs
||.golf
||.energy
||.ug
||.tn
||.report
||.np
||.mg
||.kh
||.direct
||.ad
||.tf
||.py
||.porn
||.photos
||.globo
||.bo
||.vet
||.tips
||.mu
||.movie
||.ml
||.mk
||.kw
||.ht
||.radio
||.pr
||.cu
||.bnpparibas
||.bi
||.mn
||.legal
||.gy
||.bike
||.rw
||.ps
||.ntt
||.na
||.mv
||.mt
||.je
||.ao
||.jm
||.expert
||.bh
||.zw
||.sharp
||.ci
||.cf
||.barclays
||.tz
||.style
||.sncf
||.sn
||.llc
||.jo
||.fox
||.dm
||.arpa
||.wf
||.vision
||.realtor
||.et
||.directory
||.toys
||.sm
||.sbi
||.photo
||.pharmacy
||.pg
||.ooo
||.nrw
||.mz
||.lc
||.ky
||.iq
||.film
||.dog
||.dhl
||.bn
||.af
||.vu
||.vi
||.university
||.swiss
||.rip
||.post
||.nr
||.gay
||.ga
||.dz
||.day
||.bs
||.aig
||.yt
||.sr
||.pn
||.nf
||.new
||.mom
||.mo
||.microsoft
||.ls
||.ist
||.gdn
||.gallery
||.cd
||.bj
||.bf
||.weir
||.uol
||.tj
||.tg
||.tel
||.sport
||.sl
||.pizza
||.ne
||.mov
||.loans
||.lgbt
||.leclerc
||.ki
||.how
||.hair
||.guardian
||.gle
||.glass
||.gd
||.fj
||.fish
||.dj
||.bw
||.bar
||.ax
||.amazon
||.zm
||.zip
||.va
||.tokyo
||.td
||.sony
||.science
||.saxo
||.paris
||.mw
||.mc
||.istanbul
||.gp
||.gm
||.gi
||.gal
||.cv
||.cpa
||.ck
||.cg
||.bmw
||.army
||.abb
||.vegas
||.tui
||.tours
||.solar
||.sanofi
||.sandvik
||.ngo
||.mp
||.mortgage
||.monash
||.madrid
||.luxury
||.loan
||.kpmg
||.kn
||.immo
||.hockey
||.hm
||.fo
||.dance
||.cw
||.cricket
||.boutique
||.bbva
||.bayern
||.basketball
||.barcelona
||.adult
||.abudhabi
||.ye
||.toyota
||.total
||.toshiba
||.tennis
||.taipei
||.sy
||.surf
||.study
||.sd
||.scb
||.repair
||.politie
||.pictet
||.nike
||.next
||.nc
||.landrover
||.kiwi
||.jll
||.jewelry
||.ice
||.honda
||.holiday
||.hisamitsu
||.hamburg
||.graphics
||.goo
||.gifts
||.gift
||.gent
||.ford
||.flir
||.flights
||.diet
||.crs
||.crown
||.cern
||.boo
||.bible
||.azure
||.aw
"
cat << EOF >> /tmp/inrule.txt 
$tld_list
EOF
paopao-pref -inrule /tmp/inrule.txt -outrule /tmp/force_nocn_list.txt

touch /tmp/delay.txt
while read dnsserver; do
    sed "s/{ser1}/$dnsserver/g" test_cn.yaml | sed "s/#dns_check//g" >/tmp/test_cn.yaml
    mosdns start -d /tmp -c test_cn.yaml >/dev/null 2>&1 &
    sleep 1
    delay=$(paopao-pref -server 127.0.0.1 -delay -v) && echo "$delay"",""$dnsserver" >>/tmp/delay.txt && echo "$dnsserver"": ""$delay"" ms"
    killall mosdns
done <dns_list.txt
cat /tmp/delay.txt
exit
sort -n /tmp/delay.txt | cut -d "," -f2 | head -3 >/tmp/dns_list.txt
cat /tmp/dns_list.txt
ser_num=$(cat /tmp/dns_list.txt | wc -l)
if [ "$ser_num" = "0" ]; then
    echo "no dns available."
    exit
fi
ser1=$(head -1 /tmp/dns_list.txt)
ser2=$(head -2 /tmp/dns_list.txt | tail -1)
ser3=$(tail -1 /tmp/dns_list.txt)
cp test_cn.yaml /tmp/gen_mark.yaml
if [ "$ser_num" -gt 0 ]; then
    sed -i "s/#gen_mark//g" /tmp/gen_mark.yaml
    sed -i "s/{ser1}/$ser1/g" /tmp/gen_mark.yaml
fi

if [ "$ser_num" -gt 1 ]; then
    sed -i "s/#ser_num2//g" /tmp/gen_mark.yaml
    sed -i "s/{ser2}/$ser2/g" /tmp/gen_mark.yaml
fi

if [ "$ser_num" -gt 2 ]; then
    sed -i "s/#ser_num3//g" /tmp/gen_mark.yaml
    sed -i "s/{ser3}/$ser3/g" /tmp/gen_mark.yaml
fi


mosdns start -d /tmp -c gen_mark.yaml  &
sleep 1

touch domains_ok.txt
echo "nameserver 127.0.0.1" > /etc/resolv.conf
ps
paopao-pref
cat /tmp/inrule.txt >> domains_ok.txt
paopao-pref -inrule /data/domains_ok.txt -outrule /data/global_mark.dat
xz -9 -e global_mark.dat
datsha=$(sha512sum global_mark.dat.xz |cut -d" " -f1)
echo -n $datsha > sha.txt
shasize=$(wc -c < sha.txt)
dd if=/dev/zero of=sha.txt bs=1 count=$((1024-shasize)) seek=$shasize conv=notrunc
cat global_mark.dat.xz sha.txt > global_mark.dat
sha256sum global_mark.dat| cut -d" " -f1 > /pub/global_mark.dat.sha256sum
mv global_mark.dat /pub