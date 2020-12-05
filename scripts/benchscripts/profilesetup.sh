#!/bin/bash
enableprofile=$1

if [ "$enableprofile" -eq 1 ]
then
  echo "Enabling profiling"
  sed -i 's|\t// _ "net/http/pprof"|\t_ "net/http/pprof"|' ./cmd/rpcnode/rpcnode.go
  sed -i 's|\t\t// logging.Info(http.ListenAndServe("localhost:6060", nil))|\t\tlogging.Info(http.ListenAndServe("localhost:6060", nil))|' ./cmd/rpcnode/rpcnode.go
  sed -i 's|  flags=()|  flags=(-gcflags="all=-N -l")|' ./scripts/buildgo.sh
else
  echo "Disabling profiling"
  sed -i 's|\t_ "net/http/pprof"|\t// _ "net/http/pprof"|' ./cmd/rpcnode/rpcnode.go
  sed -i 's|\t\tlogging.Info(http.ListenAndServe("localhost:6060", nil))|\t\t// logging.Info(http.ListenAndServe("localhost:6060", nil))|' ./cmd/rpcnode/rpcnode.go
  sed -i 's|  flags=(-gcflags="all=-N -l")|  flags=()|' ./scripts/buildgo.sh
fi