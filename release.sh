#!/bin/sh
go mod init paopao-perf 

go get -u

BUILD_DATE=$(date +%Y%m%d)
GIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION="${BUILD_DATE}-${GIT_HASH}"
EXT_LDFLAGS="-s -w -extldflags -static -X main.version=${VERSION} -X main.buildDate=$(date +%Y-%m-%d) -X main.gitHash=${GIT_HASH}"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_amd64_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_386_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=4_freebsd GOARCH=386 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 4_freebsd_386_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=4_freebsd GOARCH=amd64 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 4_freebsd_amd64_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=4_freebsd GOARCH=arm go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 4_freebsd_arm_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_arm_v7_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_arm_v6_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_arm_v5_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_arm64_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=mips64 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_mips64_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=mips64le go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_mips64le_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=mipsle go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_mipsle_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=linux GOARCH=mips go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 2_linux_mips_paopao-perf.tar.gz paopao-perf domains.txt

CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf.exe
tar -czvf 1_windows_386_paopao-perf.tar.gz paopao-perf.exe domains.txt

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf.exe
tar -czvf 1_windows_amd64_paopao-perf.tar.gz paopao-perf.exe domains.txt

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "${EXT_LDFLAGS}" -trimpath -o paopao-perf
tar -czvf 3_darwin_amd64_paopao-perf.tar.gz paopao-perf domains.txt

mkdir -p ./build/
mv *.tar.gz ./build/