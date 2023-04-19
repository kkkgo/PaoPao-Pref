FROM alpine:edge AS builder
ADD http://s3-us-west-1.amazonaws.com/umbrella-static/top-1m.csv.zip /src/
COPY . /src/
WORKDIR /src
RUN apk update && apk add go unzip
RUN sh /src/docker-build.sh
FROM alpine:edge
COPY --from=builder /cp/ /data/
RUN mv /data/paopao-pref /usr/bin/
WORKDIR /data/
ENTRYPOINT ["paopao-pref"]