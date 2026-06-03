#!/usr/bin/env bash
#MISE description="Build binary"
set -euo pipefail

go build -o bin/console-github-auth .
