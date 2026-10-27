#!/bin/sh
cd "$(dirname "$0")" && exec go run github.com/ajkula/ajkurig/cmd/release "$@"
