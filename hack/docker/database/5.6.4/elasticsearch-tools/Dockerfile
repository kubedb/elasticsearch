############################################
#      Image for elasticsearch dump        #
#  Image: appscode/elasticsearch:dump      #
############################################

FROM appscode/python:2.7

RUN set -x \
  && apt-get update \
  && apt-get install -y curl \
  && curl -sL https://deb.nodesource.com/setup | bash - \
  && apt-get install -y nodejs npm \
  && npm install npm -g \
  && npm install -g n \
  && n stable \
  && npm install elasticdump@3.3.1 -g \
  && ln -s /usr/bin/nodejs /usr/bin/node \
  && pip install --upgrade pip \
  && pip install pyyaml \
  && pip install elasticsearch

## osm version
ENV OSM_VERSION 0.6.1

RUN set -x \
  && curl "https://cdn.appscode.com/binaries/osm/$OSM_VERSION/osm-linux-amd64" -o /usr/local/bin/osm \
  && chmod +x /usr/local/bin/osm

COPY execute.py /execute.py
COPY utils.sh /utils.sh
RUN chmod +x /utils.sh

ENTRYPOINT ["python2.7", "execute.py"]
