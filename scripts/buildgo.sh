#!/usr/bin/env bash
fullbuild=${1:-0}

if [ "$fullbuild" -eq 1 ]
then
  flags=()
fi

set -e
set -o pipefail
echo Building: go build "${flags[@]}" ./cmd/preg/preg.go
go build "${flags[@]}" ./cmd/preg/preg.go

echo Building: go build "${flags[@]}" ./cmd/rpcbench/rpcbench.go
go build "${flags[@]}" ./cmd/rpcbench/rpcbench.go

echo Building: go build "${flags[@]}" ./cmd/rpcnode/rpcnode.go
go build "${flags[@]}" ./cmd/rpcnode/rpcnode.go

echo Building: go build "${flags[@]}" ./cmd/runcmd/runcmd.go
go build "${flags[@]}" ./cmd/runcmd/runcmd.go

echo Building: go build "${flags[@]}" ./cmd/genresults/genresults.go
go build "${flags[@]}" ./cmd/genresults/genresults.go

echo Building: go build "${flags[@]}" ./cmd/localrun/localrun.go
go build "${flags[@]}" ./cmd/localrun/localrun.go

echo Building: go build "${flags[@]}" ./cmd/gettestconsids/gettestconsids.go
go build "${flags[@]}" ./cmd/gettestconsids/gettestconsids.go
