#!/bin/sh
# apply tlds to /predata/inrule.txt
if [ -f /pub/tlds.txt ]; then
    cat /pub/tlds.txt >/tmp/inrule.txt
    echo "" >>/tmp/inrule.txt
    echo "" >>/tmp/inrule.txt
    cat /data/inrule.txt >>/tmp/inrule.txt
    paopao-pref -inrule /tmp/inrule.txt -outrule /predata/inrule.txt
fi
# apply cn to /predata/force_nocn_list.txt
if [ -f /pub/cn.txt ]; then
    cat /pub/cn.txt >/tmp/force_nocn_list.txt
    cat /data/inrule.txt >>/tmp/force_nocn_list.txt
    paopao-pref -inrule /tmp/force_nocn_list.txt -outrule /predata/force_nocn_list.txt
fi
