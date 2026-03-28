#!/bin/sh
set -eu

# --- Configuration ---
CSV_URL="http://s3-us-west-1.amazonaws.com/umbrella-static/top-1m.csv.zip"
ZIP_FILE="top-1m.csv.zip"
CSV_FILE="top-1m.csv"
DOMAINS_FILE="domains.txt"
DOMAINS_OK_FILE="domains_ok.txt"
BINARY="./paopao-pref"
MAX_RETRIES=3
MIN_ZIP_SIZE=10485760  # 10MB - zip should be at least this large
MIN_DOMAIN_COUNT=100000

# --- Helper functions ---
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

cleanup() {
    log "Cleaning up temporary files..."
    rm -f "$ZIP_FILE" "$CSV_FILE" "$BINARY" go.mod go.sum
}
trap cleanup EXIT

die() {
    log "ERROR: $*" >&2
    exit 1
}

# --- Step 1: Download with retry and validation ---
log "Downloading $ZIP_FILE ..."
download_ok=0
for i in $(seq 1 $MAX_RETRIES); do
    if curl -sLo "$ZIP_FILE" --retry 3 --retry-delay 5 --max-time 300 "$CSV_URL"; then
        # Verify file exists and is non-empty
        if [ ! -s "$ZIP_FILE" ]; then
            log "Attempt $i/$MAX_RETRIES: downloaded file is empty, retrying..."
            rm -f "$ZIP_FILE"
            sleep 5
            continue
        fi
        # Verify minimum file size
        file_size=$(wc -c < "$ZIP_FILE" | tr -d ' ')
        if [ "$file_size" -lt "$MIN_ZIP_SIZE" ]; then
            log "Attempt $i/$MAX_RETRIES: file too small (${file_size} bytes), retrying..."
            rm -f "$ZIP_FILE"
            sleep 5
            continue
        fi
        download_ok=1
        break
    else
        log "Attempt $i/$MAX_RETRIES: curl failed (exit code $?), retrying..."
        rm -f "$ZIP_FILE"
        sleep 5
    fi
done
[ "$download_ok" -eq 1 ] || die "Failed to download $ZIP_FILE after $MAX_RETRIES attempts"
log "Download complete ($(wc -c < "$ZIP_FILE" | tr -d ' ') bytes)"

# --- Step 2: Extract and validate CSV ---
log "Extracting $ZIP_FILE ..."
if ! unzip -o "$ZIP_FILE"; then
    die "Failed to extract $ZIP_FILE"
fi
[ -f "$CSV_FILE" ] || die "$CSV_FILE not found after extraction"

csv_lines=$(wc -l < "$CSV_FILE" | tr -d ' ')
log "Extracted $csv_lines lines from $CSV_FILE"
[ "$csv_lines" -gt 0 ] || die "$CSV_FILE is empty"

cut -d"," -f2 "$CSV_FILE" > "$DOMAINS_FILE"
domain_count=$(wc -l < "$DOMAINS_FILE" | tr -d ' ')
log "Extracted $domain_count domains"
[ "$domain_count" -gt 0 ] || die "No domains extracted from $CSV_FILE"

rm -f "$ZIP_FILE" "$CSV_FILE"

# --- Step 3: Build Go binary ---
log "Installing Go and building binary..."
sudo apt-get -qq -y install golang || die "Failed to install golang"

if [ ! -f go.mod ]; then
    go mod init paopao-perf || die "go mod init failed"
fi
go mod tidy || die "go mod tidy failed"
go build -ldflags "-s -w" -trimpath -o "$BINARY" || die "go build failed"
[ -x "$BINARY" ] || chmod +x "$BINARY"
log "Build complete"

# --- Step 4: Run DNS check ---
export FILE_OUTPUT=yes
export DNS_LIMIT=15
export DNS_SLEEP=0ms
export DNS_TIMEOUT=3s
touch "$DOMAINS_OK_FILE"

log "Running DNS check..."
if ! "$BINARY"; then
    die "paopao-pref binary exited with error"
fi

# --- Step 5: Validate results and update ---
count=$(wc -l < "$DOMAINS_OK_FILE" | tr -d ' ')
log "DNS check complete: $count valid domains found"

if [ "$count" -gt "$MIN_DOMAIN_COUNT" ]; then
    mv "$DOMAINS_OK_FILE" "$DOMAINS_FILE"
    log "Updated $DOMAINS_FILE with $count domains"
else
    log "WARNING: Only $count domains passed (threshold: $MIN_DOMAIN_COUNT), keeping existing $DOMAINS_FILE"
    rm -f "$DOMAINS_OK_FILE"
fi

log "Done"
