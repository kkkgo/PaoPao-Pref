FROM alpine:edge AS builder
RUN apk add --no-cache go git
RUN git clone https://github.com/kkkgo/PaoPao-Pref.git /src
WORKDIR /src
RUN version=$(git describe --tags --always 2>/dev/null || echo "unknown") && \
    buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
    gitHash=$(git rev-parse --short HEAD) && \
    go mod init paopao-pref && go get -u && go build -ldflags "-s -w -X main.version=${version} -X main.buildDate=${buildDate} -X main.gitHash=${gitHash}" -trimpath -o /paopao-pref

FROM sliamb/paopaodns AS download
RUN apk add --no-cache curl git
COPY --from=builder /paopao-pref /usr/bin/paopao-pref
WORKDIR /data
COPY download.sh /data/
COPY * /predata/
RUN sh /data/download.sh

FROM alpine:edge
RUN apk update && \
    apk upgrade && \
    apk add --no-cache xz && \
    rm -rf /var/cache/apk/*
COPY --from=builder /paopao-pref /usr/bin/paopao-pref
COPY --from=download /data/CN-local.dat /data/
COPY --from=download /data/cn.rules /data/
COPY --from=download /data/global.cnfilter.rules /data/
COPY --from=download /data/global.rules /data/
COPY --from=download /data/alreadymark.skip.rules /data/
COPY --from=download /data/topdomains.txt /data/
COPY --from=download /data/topdomains.rules /data/
COPY dns_list.txt /data/
COPY mark_update.sh /data/
ENV TZ=Asia/Shanghai \
    DNS_SERVER="" \
    DNS_PORT="" \
    DNS_LINE="" \
    DNS_LIMIT="25" \
    DNS_TIMEOUT="3s" \
    DNS_SLEEP="0ms" \
    FILE_OUTPUT=yes
WORKDIR /data/
CMD sh mark_update.sh
