FROM node:8.9-alpine

RUN npm install elasticdump@3.3.1 -g

COPY osm /usr/local/bin/osm
COPY elasticsearch-tools.sh /usr/local/bin/elasticsearch-tools.sh

ENTRYPOINT ["elasticsearch-tools.sh"]
