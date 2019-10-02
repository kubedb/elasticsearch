FROM node:10-alpine

RUN set -x \
  && apk add --update --no-cache bash ca-certificates

RUN npm install elasticdump@v6.11.2 -g

COPY osm /usr/local/bin/osm
COPY elasticsearch-tools.sh /usr/local/bin/elasticsearch-tools.sh

ENTRYPOINT ["elasticsearch-tools.sh"]
