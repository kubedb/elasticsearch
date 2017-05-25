#!/usr/bin/env bash

kubectl delete service elasticsearch-demo,governing-elasticsearch
kubectl delete statefulset k8sdb-elasticsearch-demo
