#########################################
#      Image for Elasticsearch          #
#  Image: appscode/elasticsearch:<tag>  #
#########################################

FROM appscode/oracle-jdk:8

## elasticsearch version
ENV ES_PKG_NAME elasticsearch-2.3.1

## gosu version
ENV GOSU_VERSION 1.7

## es user to run elasticsearch
RUN groupadd -r es --gid=999 && useradd -r -g es --uid=999 es

##### Install ElasticSearch ######
RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates wget gettext-base\
  && wget https://download.elasticsearch.org/elasticsearch/elasticsearch/$ES_PKG_NAME.tar.gz \
  && tar xvzf $ES_PKG_NAME.tar.gz \
  && rm -f $ES_PKG_NAME.tar.gz \
  && mv /$ES_PKG_NAME /elasticsearch \
##################################
########## install gosu ##########
  && wget -O /usr/local/bin/gosu "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$(dpkg --print-architecture)" \
  && wget -O /usr/local/bin/gosu.asc "https://github.com/tianon/gosu/releases/download/$GOSU_VERSION/gosu-$(dpkg --print-architecture).asc" \
  && export GNUPGHOME="$(mktemp -d)" \
  && gpg --keyserver ha.pool.sks-keyservers.net --recv-keys B42F6819007F00F88E364FD4036A9C25BF357DD4 \
  && gpg --batch --verify /usr/local/bin/gosu.asc /usr/local/bin/gosu \
  && rm -r "$GNUPGHOME" /usr/local/bin/gosu.asc \
  && chmod +x /usr/local/bin/gosu \
  && gosu nobody true
##################################

###### runit service ######
COPY lib/sv /etc/sv
RUN ln -s /etc/sv /etc/service
RUN chmod +x /etc/service/*/run
###########################

## elasticsearch config file
COPY lib/elasticsearch.yml /elasticsearch/config/template.yml

VOLUME ["/var/pv"]

EXPOSE 9200
EXPOSE 9300
ENTRYPOINT ["/runit.sh"]
