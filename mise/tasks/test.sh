#!/usr/bin/env bash
#MISE description="Run Go tests with the race detector"
set -euo pipefail

go test -race ./...
