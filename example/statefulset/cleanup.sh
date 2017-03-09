#!/usr/bin/env bash

kubectl delete sa governing-elasticsearch
kubectl delete service elasticsearch-demo
kubectl delete statefulset db-elasticsearch-demo
