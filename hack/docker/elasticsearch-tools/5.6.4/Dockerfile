FROM alpine:3.6

RUN apk add --update nodejs nodejs-npm bash \
    && rm -rf /var/cache/apk/* \
    && npm install -g elasticdump@3.3.1

COPY osm /usr/local/bin/osm
COPY elasticsearch-tools.sh /usr/local/bin/elasticsearch-tools.sh

ENTRYPOINT ["elasticsearch-tools.sh"]
