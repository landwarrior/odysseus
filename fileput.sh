#!/usr/bin/env bash

set -e

mkdir -p /var/odysseus/myapp
# 削除は必要な時だけにする
# rm -fr /var/odysseus/myapp/*
cp -fr /vagrant_data/myapp /var/odysseus
