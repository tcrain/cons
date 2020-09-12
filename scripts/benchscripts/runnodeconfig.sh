#!/bin/bash
set -e
set -o pipefail

nodecounts=$1
benchid=$2
tofilenames=$3
ipfile=$4
pregip=$5
tofolder=$6

for nc in $nodecounts
do

    IFS="," read -a arr <<< $nc
    echo
    bash ./scripts/benchscripts/updatetestconfigs.sh NumTotalProcs "${arr[0]}" "$tofolder"

    if [ "${#arr[@]}" -ge 2 ]
    then
        bash ./scripts/benchscripts/updatetestconfigs.sh FanOut "${arr[1]}" "$tofolder"
    fi

    if [ "${#arr[@]}" -ge 3 ]
    then
        bash ./scripts/benchscripts/updatetestconfigs.sh RndMemberCount "${arr[2]}" "$tofolder"
    fi
    echo

    bash ./scripts/benchscripts/runtestconfigs.sh "$benchid" "$tofilenames" "$ipfile" "$pregip" "$tofolder" "$nc"
done
