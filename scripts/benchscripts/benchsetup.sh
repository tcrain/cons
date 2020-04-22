set -e
set -o pipefail

user=$1
ipfile=$2
key=$3
pregip=$4
pregport=$5
benchfolder=$6
nwtest=$7
tofolder=$8

echo "Running: mkdir $benchfolder"
mkdir -p "$benchfolder"

echo "Building"
bash ./scripts/buildgo.sh 1
bash ./scripts/benchscripts/resetconfigs.sh "$tofolder"

echo "Starting nodes"
if [ "$nwtest" -eq 1 ]
then
    bash ./scripts/setupnodes.sh "$user" "$ipfile" "$key" "$pregip" "$pregport"
else
    bash ./scripts/setuprpc.sh
fi

sleep 3
