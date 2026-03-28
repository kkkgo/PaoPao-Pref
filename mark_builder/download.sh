#!/bin/sh
set -e

MAX_RETRIES=3
RETRY_DELAY=5

log() { echo "[$(date '+%H:%M:%S')] $*"; }
die() { log "ERROR: $*" >&2; exit 1; }

retry() {
	local n=0
	until [ $n -ge $MAX_RETRIES ]; do
		"$@" && return 0
		n=$((n + 1))
		log "Retry $n/$MAX_RETRIES: $*"
		sleep $RETRY_DELAY
	done
	die "Failed after $MAX_RETRIES retries: $*"
}

check_size() {
	local file="$1" min="$2" label="$3"
	local size
	size=$(wc -c <"$file")
	if [ "$size" -gt "$min" ]; then
		log "$label validated ($size bytes)"
	else
		die "$label size too small: $size (min: $min)"
	fi
}

check_lines() {
	local file="$1" min="$2" label="$3"
	local lines
	lines=$(wc -l <"$file")
	if [ "$lines" -gt "$min" ]; then
		log "$label validated ($lines lines)"
	else
		die "$label line count too low: $lines (min: $min)"
	fi
}

mkdir -p /data

# ============================================================
# 1. Download CN-local.dat with sha256 verification
# ============================================================
log "Downloading CN-local.dat..."
retry git clone --depth 1 -b main https://github.com/kkkgo/Country-only-cn-private.mmdb /data/mmdb_repo
cp /data/mmdb_repo/CN-local.dat /data/CN-local.dat
dat_hash=$(sha256sum /data/CN-local.dat | cut -d" " -f1)
dat_expected=$(cat /data/mmdb_repo/CN-local.dat.sha256sum | tr -d '[:space:]')
rm -rf /data/mmdb_repo
if [ "$dat_hash" = "$dat_expected" ]; then
	log "CN-local.dat hash verified"
else
	die "CN-local.dat hash mismatch: expected=$dat_expected got=$dat_hash"
fi
check_size /data/CN-local.dat 190000 "CN-local.dat"

# ============================================================
# 2. Download topdomains: /data/topdomains.data
# ============================================================
log "Downloading topdomains..."
retry curl -sLo /data/top-1m.csv.zip http://s3-us-west-1.amazonaws.com/umbrella-static/top-1m.csv.zip
unzip -o /data/top-1m.csv.zip -d /data
cut -d"," -f2 /data/top-1m.csv >/data/topdomains.data
check_size /data/topdomains.data 20000000 "topdomains.data"

# ============================================================
# 3. Download proxy rules: /data/global.nofilter.rules
# ============================================================
log "Downloading proxy rules (gfwlist)..."
retry git clone --depth 1 -b master https://github.com/gfwlist/gfwlist /data/gfwlist_repo
cp /data/gfwlist_repo/list.txt /data/proxy.rules.txt
rm -rf /data/gfwlist_repo
check_lines /data/proxy.rules.txt 4000 "proxy.rules.txt"

echo "" >>/data/proxy.rules.txt
if [ -f /predata/global.hook.rules ]; then
	cat /predata/global.hook.rules >>/data/proxy.rules.txt
fi
paopao-pref -inrule /data/proxy.rules.txt -outrule /data/global.nofilter.rules

# ============================================================
# 4. Download CN rules: /data/cn.rules
# ============================================================
log "Downloading CN rules (v2ray direct-list)..."
retry git clone --depth 1 -b release https://github.com/Loyalsoldier/v2ray-rules-dat /data/v2ray_repo
cp /data/v2ray_repo/direct-list.txt /data/cn.txt
rm -rf /data/v2ray_repo
check_size /data/cn.txt 50000 "cn.txt"

if [ -f /predata/cn_mark.rules ]; then
	echo "" >>/data/cn.txt
	cat /predata/cn_mark.rules >>/data/cn.txt
fi
paopao-pref -inrule /data/cn.txt -outrule /data/cn.rules

# ============================================================
# 5. Generate global.cnfilter.rules: cn.hook.rules + cn.rules
# ============================================================
log "Generating global.cnfilter.rules..."
if [ -f /predata/cn.hook.rules ]; then
	cat /predata/cn.hook.rules >/data/cn.hook.raw
fi
touch /data/cn.hook.raw
echo "" >>/data/cn.hook.raw
cat /data/cn.rules >>/data/cn.hook.raw
echo "" >>/data/cn.hook.raw
paopao-pref -inrule /data/cn.hook.raw -outrule /data/global.cnfilter.rules

# ============================================================
# 6. Generate global.rules: global.nofilter.rules - global.cnfilter.rules
# ============================================================
log "Generating global.rules..."
paopao-pref -inrule /data/global.nofilter.rules -filter /data/global.cnfilter.rules -outrule /data/global.rules

# ============================================================
# 7. Generate alreadymark.skip.rules: global.rules + global.cnfilter.rules
# ============================================================
log "Generating alreadymark.skip.rules..."
touch /data/skip.raw
echo "" >>/data/skip.raw
cat /data/global.rules >>/data/skip.raw
echo "" >>/data/skip.raw
cat /data/global.cnfilter.rules >>/data/skip.raw
echo "" >>/data/skip.raw
paopao-pref -inrule /data/skip.raw -outrule /data/alreadymark.skip.rules

# ============================================================
# 8. Generate topdomains.rules
# ============================================================
log "Generating topdomains.rules..."
cp /data/topdomains.data /data/topdomains.txt
echo "" >>/data/topdomains.data
cat /data/alreadymark.skip.rules >>/data/topdomains.data
echo "" >>/data/topdomains.data
paopao-pref -inrule /data/topdomains.data -outrule /data/topdomains.rules

log "Download and processing completed."
