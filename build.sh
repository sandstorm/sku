#!/bin/bash

set -ex

go build cmd/sku/sku.go
go build -o sku_plugins/sandstorm.so -buildmode=plugin plugin/sandstorm/sandstorm_plugin.go