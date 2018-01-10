#!/bin/bash

searchguard="/elasticsearch/plugins/search-guard-5"
certs="/elasticsearch/config/certs"

sync

until curl -s 'https://localhost:9200' --insecure > /dev/null
do
    sleep 0.1
done

"$searchguard"/tools/sgadmin.sh \
    -ks "$certs"/sgadmin.jks \
    -ts "$certs"/truststore.jks \
    -cd "$searchguard"/sgconfig -icl -nhnv
