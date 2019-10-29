# Copyright The KubeDB Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM openjdk:8u181-alpine

ENV ES_VERSION 6.8.0

ENV DOWNLOAD_URL "https://artifacts.elastic.co/downloads/elasticsearch"
ENV ES_TARBAL "${DOWNLOAD_URL}/elasticsearch-${ES_VERSION}.tar.gz"
ENV ES_TARBALL_ASC "${DOWNLOAD_URL}/elasticsearch-${ES_VERSION}.tar.gz.asc"
ENV GPG_KEY "46095ACC8548582C1A2699A9D27D666CD88E42B4"

# Install necessary tools
RUN apk add --no-cache --update bash ca-certificates su-exec util-linux curl runit

# Install Elasticsearch.
RUN apk add --no-cache -t .build-deps gnupg openssl \
  && cd /tmp \
  && echo "===> Install Elasticsearch..." \
  && curl -o elasticsearch.tar.gz -Lskj "$ES_TARBAL"; \
	if [ "$ES_TARBALL_ASC" ]; then \
		curl -o elasticsearch.tar.gz.asc -Lskj "$ES_TARBALL_ASC"; \
		export GNUPGHOME="$(mktemp -d)"; \
		gpg --keyserver ha.pool.sks-keyservers.net --recv-keys "$GPG_KEY"; \
		gpg --batch --verify elasticsearch.tar.gz.asc elasticsearch.tar.gz; \
		rm -r "$GNUPGHOME" elasticsearch.tar.gz.asc; \
	fi; \
  tar -xf elasticsearch.tar.gz \
  && ls -lah \
  && mv elasticsearch-$ES_VERSION /elasticsearch \
  && adduser -DH -s /sbin/nologin elasticsearch \
  && mkdir -p /elasticsearch/config/scripts /elasticsearch/plugins \
  && chown -R elasticsearch:elasticsearch /elasticsearch \
  && rm -rf /tmp/* \
  && apk del --purge .build-deps

# Add Elasticsearch to PATH
ENV PATH /elasticsearch/bin:$PATH

# Set default working directory to /elasticsearch
WORKDIR /elasticsearch

# Set environment variables defaults
ENV NODE_NAME="" \
    ES_TMPDIR="/elasticsearch/tmp" \
    ES_JAVA_OPTS="-Xms512m -Xmx512m" \
    CLUSTER_NAME="elasticsearch" \
    NODE_MASTER=true \
    NODE_DATA=true \
    NODE_INGEST=true \
    HTTP_ENABLE=true \
    HTTP_CORS_ENABLE=true \
    HTTP_CORS_ALLOW_ORIGIN=* \
    DISCOVERY_SERVICE="" \
    NUMBER_OF_MASTERS=1 \
    SSL_ENABLE=false \
    MODE="" \
    NETWORK_HOST=_site_ \
    HTTP_CORS_ENABLE=true \
    MAX_LOCAL_STORAGE_NODES=1 \
    SHARD_ALLOCATION_AWARENESS="" \
    SHARD_ALLOCATION_AWARENESS_ATTR="" \
    MEMORY_LOCK=true \
    REPO_LOCATIONS="" \
    KEY_PASS=""

# Install mapper-attachments (https://www.elastic.co/guide/en/elasticsearch/plugins/current/ingest-attachment.html)
RUN ./bin/elasticsearch-plugin install --batch ingest-attachment

# Install search-guard
RUN ./bin/elasticsearch-plugin install --batch -b com.floragunn:search-guard-6:6.8.0-25.4
RUN chmod +x -R plugins/search-guard-6/tools/*.sh

# Add Elasticsearch configuration files
ADD config /elasticsearch/config

# run.sh run Elasticsearch after changing ownership and some configuration
COPY run.sh /
RUN mkdir /etc/service/elasticsearch
RUN ln -s /run.sh /etc/service/elasticsearch/run

# fsloader watcher for any configuration changes and restart Elasticsearch if necessary
ADD fsloader /fsloader
RUN chmod +x /fsloader/*
RUN mkdir /etc/service/fsloader
RUN ln -s /fsloader/run_fsloader.sh /etc/service/fsloader/run

# /elasticsearch/config/certs directory is used to mount SSL certificates
RUN mkdir /elasticsearch/config/certs
RUN chown elasticsearch:elasticsearch -R /elasticsearch/config/certs

# yq and config-marger.sh is used to merge custom configuration files.
COPY yq /usr/bin/yq
COPY config-merger.sh /usr/bin/config-merger.sh

# runit.sh run at Entrypoint
COPY runit.sh /runit.sh

# Volume for Elasticsearch data
VOLUME ["/data"]

# Export HTTP & Transport
EXPOSE 9200 9300

# Run "runit.sh" on start
ENTRYPOINT ["/runit.sh"]
