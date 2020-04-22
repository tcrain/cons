#!/bin/bash
#set -e
set -o pipefail

echo Testing bincons1
go test -v -timeout=30m ./consensus/cons/bincons1/... > bincons1.out 2>&1
echo $?

echo Testing binconsrnd1
go test -v -timeout=30m ./consensus/cons/binconsrnd1/... > binconsrnd1.out 2>&1
echo $?

echo Testing mvcons1
go test -v -timeout=30m ./consensus/cons/mvcons1/... > mvcons1.out 2>&1
echo $?

echo Testing mvcons2
go test -v -timeout=30m ./consensus/cons/mvcons2/... > mvcons2.out 2>&1
echo $?

echo Testing mvcons3
go test -v -timeout=30m ./consensus/cons/mvcons3/... > mvcons3.out 2>&1
echo $?

echo Testing rbbcast1
go test -v -timeout=30m ./consensus/cons/rbbcast1/... > rbbcast1.out 2>&1
echo $?

echo Testing rbbcast2
go test -v -timeout=30m ./consensus/cons/rbbcast2/... > rbbcast2.out 2>&1
echo $?

echo "Success test"
