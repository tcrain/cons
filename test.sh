#!/bin/bash
set -e
set -o pipefail

cd internal

echo "Unit test"

# Test auth
cd auth
go test -v -race -timeout=30m ./...
cd ..

# Test messsages
cd messages
go test -v -race -timeout=30m ./...
cd ..

# Test messsagetypes
cd messagetypes
go test -v -race -timeout=30m ./...
cd ..

# Test storage
cd storage
go test -v -race -timeout=30m ./...
cd ..

# Test channel
cd channel
go test -v -race -timeout=30m ./...
cd ..

# Test channelinterface
cd channelinterface
go test -v -race -timeout=30m ./...
cd ..

# Test consinterface
cd consinterface
go test -v -race -timeout=30m ./...
cd ..

# Test logging
cd logging
go test -v -race -timeout=30m ./...
cd ..

# Test utils
cd utils
go test -v -race -timeout=30m ./...
cd ..

# Test network
cd network
go test -v -race -timeout=30m ./...
cd ..

# Test cons
cd cons
go test -v -timeout=30m -test.run=TestBinCons ./...
go test -v -timeout=30m -test.run=TestMvCons ./...
go test -v -timeout=30m -test.run=TestSimpleCons ./...
cd ..

# Benchmarks
# - go test -bench BenchmarkSimpleCons -benchtime 10s -test.run none ./...
# - go test -bench BenchmarkBinCons1 -benchtime 10s -test.run none ./...
go test -bench BenchmarkSimpleCons -test.run none ./...
go test -bench BenchmarkBinCons1 -test.run none ./...
go test -bench BenchmarkStorage -test.run none ./...
#- go test -v ./...
