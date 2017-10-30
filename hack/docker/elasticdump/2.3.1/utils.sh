#!/bin/bash

exec 1> >(logger -s -p daemon.info -t es)
exec 2> >(logger -s -p daemon.error -t es)

RETVAL=0

backup(){
  # 1 - host
  # 2 - snapshot-name
  # 3 - index

  path=/var/dump-backup/$2/$3
  mkdir -p $path
  cd $path
  rm -rf $path/*

  elasticdump --quiet --input http://$1:9200/$3 --output $3.mapping.json --type mapping
  retval=$?
  if [ "$retval" -ne 0 ]; then
    echo "Fail to dump mapping for $3"
    exit 1
  fi

  elasticdump --quiet --input http://$1:9200/$3 --output $3.analyzer.json --type analyzer
  retval=$?
  if [ "$retval" -ne 0 ]; then
    echo "Fail to dump analyzer for $3"
    exit 1
  fi

  elasticdump --quiet --input http://$1:9200/$3 --output $3.json --type data
  retval=$?
  if [ "$retval" -ne 0 ]; then
    echo "Fail to dump data for $3"
    exit 1
  fi

  echo "Successfully dump for $3"
}

restore(){
  # 1 - host
  # 2 - snapshot-name
  # 3 - index
  path=/var/dump-restore/$2/$3
  cd $path

  elasticdump --quiet --input $3.analyzer.json --output http://$1:9200/$3 --type analyzer
  retval=$?
  if [ "$retval" -ne 0 ]; then
    echo "Fail to restore analyzer for $3"
    exit 1
  fi

  elasticdump --quiet --input $3.mapping.json --output http://$1:9200/$3 --type mapping
  retval=$?
  if [ "$retval" -ne 0 ]; then
    echo "Fail to restore mapping for $3"
    exit 1
  fi


  elasticdump --quiet --input $3.json --output http://$1:9200/$3 --type data
  retval=$?
  if [ "$retval" -ne 0 ]; then
    echo "Fail to restore data for $3"
    exit 1
  fi

  echo "Successfully restore for $3"
}

push() {
  # 1 - bucket
  # 2 - folder
  # 3 - snapshot-name

  src_path="/var/dump-backup/$3"
  osm push --osmconfig=/etc/osm/config -c "$1" "$src_path" "$2/$3"
  retval=$?
  if [ "$retval" -ne 0 ]; then
        exit 1
  fi

  exit 0
}

pull() {
  # 1 - bucket
  # 2 - folder
  # 3 - snapshot-name

  dst_path="/var/dump-restore/$3"
  mkdir -p "$dst_path"
  rm -rf "$dst_path"

  osm pull --osmconfig=/etc/osm/config -c "$1" "$2/$3" "$dst_path"
  retval=$?
  if [ "$retval" -ne 0 ]; then
        exit 1
  fi

  exit 0
}


process=$1
shift
case "$process" in
	backup)
		backup "$@"
		;;
	restore)
		restore "$@"
		;;
	push)
	  push "$@"
	  ;;
	pull)
		pull "$@"
		;;
	*)	(10)
		echo $"Unknown process!"
		RETVAL=1
esac
exit "$RETVAL"
