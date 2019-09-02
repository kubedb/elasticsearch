FROM node:8.14-alpine

RUN set -x \
  && apk add --update --no-cache bash ca-certificates

RUN npm install elasticdump@4.1.2 -g

COPY osm /usr/local/bin/osm
COPY elasticsearch-tools.sh /usr/local/bin/elasticsearch-tools.sh

ENTRYPOINT ["elasticsearch-tools.sh"]
