#!/bin/sh
# apply tlds
cat /pub/tlds.txt >/tmp/force_list_global.txt
echo "" >>/tmp/force_list_global.txt
cat /data/fwrule.txt >>/tmp/force_list_global.txt
echo "" >>/tmp/force_list_global.txt
cat /pub/cn.txt >>/tmp/force_list_global.txt
echo "" >>/tmp/force_list_global.txt
paopao-pref -inrule /tmp/force_list_global.txt -outrule /predata/force_list_global.txt
paopao-pref -inrule /pub/tlds.txt -outrule /predata/tlds.txt

# apply cn
cat /pub/cn.txt >/tmp/force_list_cn.txt
echo "" >>/tmp/force_list_cn.txt
cat /data/fwrule.txt >>/tmp/force_list_cn.txt
echo "" >>/tmp/force_list_cn.txt
paopao-pref -inrule /tmp/force_list_cn.txt -outrule /predata/force_list_cn.txt
paopao-pref -inrule /pub/cn.txt -outrule /predata/cn.txt

# apply domains
cat /data/fwrule.txt >/tmp/pfdata.txt
echo "" >>/tmp/pfdata.txt
cat /data/domains.txt >>/tmp/pfdata.txt
echo "" >>/tmp/pfdata.txt
cat /pub/tlds.txt >>/tmp/pfdata.txt
echo "" >>/tmp/pfdata.txt
cat /pub/cn.txt >>/tmp/pfdata.txt
paopao-pref -inrule /tmp/pfdata.txt -outrule /predata/pfdata.txt
