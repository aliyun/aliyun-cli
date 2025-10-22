#!/bin/bash

# scripts to generate log.pb.go 
# use protoc and protoc-gen-gogo

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "start generating log.pb.go..."

if ! command -v protoc &> /dev/null; then
    echo "error protoc not found. Please install Protocol Buffers Compiler first"
    echo "macOS: brew install protobuf"
    echo "Linux: apt-get install protobuf-compiler 或 yum install protobuf-compiler"
    exit 1
fi

if [ -z "$GOPATH" ]; then
    GOPATH=$(go env GOPATH)
fi

export PATH="$GOPATH/bin:$PATH"

if ! command -v protoc-gen-gogo &> /dev/null; then
    echo "protoc-gen-gogo installing..."
    go install github.com/gogo/protobuf/protoc-gen-gogo@latest
fi

protoc --gogo_out=. logs.proto

if [ -f "logs.pb.go" ]; then
    echo "generate logs.pb.go success"
else
    echo "warning: generate log.pb.go failed"
    exit 1
fi

echo "Done"

