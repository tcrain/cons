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
    echo
    bash ./scripts/benchscripts/updatetestconfigs.sh NumTotalProcs $nc "$tofolder"

    bash ./scripts/benchscripts/runtestconfigs.sh "$benchid" "$tofilenames" "$ipfile" "$pregip" "$tofolder" "$nc"
done
