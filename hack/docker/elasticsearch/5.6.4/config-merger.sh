#!/bin/bash
set -x
CONFIG_DIR="/elasticsearch/config"
CUSTOM_CONFIG_DIR="/elasticsearch/custom-config"

CONFIG_FILE=$CONFIG_DIR/elasticsearch.yml

# if common-config file exist then apply it
if [ -f $CUSTOM_CONFIG_DIR/common-config.yaml ]; then
	yq merge -i --overwrite $CONFIG_FILE $CUSTOM_CONFIG_DIR/common-config.yaml
fi

# if it is data node and data-config file exist then apply it
if [[ ("$NODE_DATA" = true ) && (-f $CUSTOM_CONFIG_DIR/data-config.yaml) ]]; then
	yq merge -i --overwrite $CONFIG_FILE $CUSTOM_CONFIG_DIR/data-config.yaml
fi

# if it is client node and client-config file exist then apply it
if [[ ("$NODE_INGEST" = true ) && ("$MODE" = client) && (-f $CUSTOM_CONFIG_DIR/client-config.yaml) ]]; then
		yq merge -i --overwrite $CONFIG_FILE $CUSTOM_CONFIG_DIR/client-config.yaml
fi

# if it is master node and mater-config file exist then apply it
if [[ ("$NODE_MASTER" = true ) && (-f $CUSTOM_CONFIG_DIR/master-config.yaml) ]]; then
	yq merge -i --overwrite $CONFIG_FILE $CUSTOM_CONFIG_DIR/master-config.yaml
fi
