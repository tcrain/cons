#!/bin/bash

#CONTYPE="UDP"
#CONTYPE="TCP"
CONTYPE="P2p"

#CSTYPE="SimpleCons"
CSTYPE="BinCons1"
#CSTYPE="MvCons1"

MEMSTORE=""
#MEMSTORE="Memstore"

cd internal/cons

case "$1" in
    
    cpu) go test -v -test.run=TestProfile${CSTYPE}${MEMSTORE}${CONTYPE} ./... -cpuprofile cpu.out
	 go tool pprof --pdf ./cons.test ./cpu.out > cpu.pdf
	 go tool pprof cons.test cpu.out
	 ;;
	 
    mem) go test -v -test.run=TestProfile${CSTYPE}${MEMSTORE}${CONTYPE} ./... -memprofile mem.out -test.memprofilerate=1
	 go tool pprof --alloc_space --pdf ./cons.test ./mem.out > mem.pdf
	 go tool pprof --alloc_space cons.test mem.out
	 ;;
	 
    mutex) go test -v -test.run=TestProfile${CSTYPE}${MEMSTORE}${CONTYPE} ./... -mutexprofile mutex.out
	   go tool pprof --pdf ./cons.test ./mutex.out > mutex.pdf
	   go tool pprof cons.test mutex.out
	   ;;
		
    block) go test -v -test.run=TestProfile${CSTYPE}${MEMSTORE}${CONTYPE} ./... -blockprofile block.out
	   go tool pprof --pdf ./cons.test ./block.out > block.pdf
	   go tool pprof cons.test block.out
	   ;;
    
    *) echo "Invalid profile type"
       ;;
esac
