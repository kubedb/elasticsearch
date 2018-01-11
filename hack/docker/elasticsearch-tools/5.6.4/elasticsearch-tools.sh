#!/bin/bash
set -eou pipefail

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

show_help() {
    echo "elasticsearch-tools.sh - run tools"
    echo " "
    echo "elasticsearch-tools.sh COMMAND [options]"
    echo " "
    echo "options:"
    echo "-h, --help                         show brief help"
    echo "    --data-dir=DIR                 path to directory holding db data (default: /var/data)"
    echo "    --host=HOST                    database host"
    echo "    --user=USERNAME                database username"
    echo "    --indices=INDICES              elasticsearch indices"
    echo "    --bucket=BUCKET                name of bucket"
    echo "    --folder=FOLDER                name of folder in bucket"
    echo "    --snapshot=SNAPSHOT            name of snapshot"
}

RETVAL=0
DEBUG=${DEBUG:-}
DB_HOST=${DB_HOST:-}
DB_PORT=${DB_PORT:-9200}
DB_USER=${DB_USER:-}
DB_PASSWORD=${DB_PASSWORD:-}
DB_INDICES=${DB_INDICES:-}
DB_BUCKET=${DB_BUCKET:-}
DB_FOLDER=${DB_FOLDER:-}
DB_SNAPSHOT=${DB_SNAPSHOT:-}
DB_DATA_DIR=${DB_DATA_DIR:-/var/data}
OSM_CONFIG_FILE=/etc/osm/config

op=$1
shift

while test $# -gt 0; do
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        --data-dir*)
            export DB_DATA_DIR=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --host*)
            export DB_HOST=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --user*)
            export DB_USER=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --indices*)
            export DB_INDICES=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --bucket*)
            export DB_BUCKET=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --folder*)
            export DB_FOLDER=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --snapshot*)
            export DB_SNAPSHOT=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
done

if [ -n "$DEBUG" ]; then
    env | sort | grep DB_*
    echo ""
fi

# cleanup data dump dir
mkdir -p "$DB_DATA_DIR"
cd "$DB_DATA_DIR"
rm -rf *

export NODE_TLS_REJECT_UNAUTHORIZED=0

function exit_on_error() {
    echo "$1"
    exit 1
}

# Wait for elasticsearch to start
# ref: http://unix.stackexchange.com/a/5279
echo $DB_HOST $DB_PORT
while ! nc $DB_HOST $DB_PORT -w 30 > /dev/null; do echo "Waiting... database is not ready yet"; sleep 5; done

case "$op" in
    backup)
        for index in $(echo "$DB_INDICES" | sed "s/,/ /g")
        do
            elasticdump --quiet --input "https://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$index" --output "$index.mapping.json" --type mapping || exit_on_error "failed to dump mapping for $index"
            elasticdump --quiet --input "https://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$index" --output "$index.analyzer.json" --type analyzer || exit_on_error "failed to dump analyzer for $index"
            elasticdump --quiet --input "https://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$index" --output "$index.data.json" --type data || exit_on_error "failed to dump data for $index"

            echo "$index" >> indices.txt
        done

        osm push --osmconfig="$OSM_CONFIG_FILE" -c "$DB_BUCKET" "$DB_DATA_DIR" "$DB_FOLDER/$DB_SNAPSHOT" || exit_on_error "failed to push data"
        ;;
    restore)
        osm pull --osmconfig="$OSM_CONFIG_FILE" -c "$DB_BUCKET" "$DB_FOLDER/$DB_SNAPSHOT" "$DB_DATA_DIR" || exit_on_error "failed to pull data"

        IFS=$'\n'
        for index in $(cat indices.txt)
        do
            elasticdump --quiet --input "$index.analyzer.json" --output "https://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$index" --type analyzer || exit_on_error "failed to restore analyzer for $index"
            elasticdump --quiet --input "$index.mapping.json" --output "https://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$index" --type mapping || exit_on_error "failed to restore mapping for $index"
            elasticdump --quiet --input "$index.data.json" --output "https://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$index" --type data || exit_on_error "failed to restore data for $index"
        done
        ;;
    *)  (10)
        echo $"Unknown op!"
        RETVAL=1
esac
exit "$RETVAL"
