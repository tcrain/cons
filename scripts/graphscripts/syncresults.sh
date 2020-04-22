set -e
set -o pipefail

if [ "$#" -lt 1 ]
then
    echo "This command sycnronizes the bench results folder with a remote node"
    echo "Usage: \"./scripts/graphscripts/syncresults.sh ip user keyfile"
    exit
fi

ip=$1
user=${2:-$BENCHUSER}
key=${3:-$KEYPATH}

# resultsPath="~/go/src/github.com/tcrain/cons/benchresults"
resultsArchive="benchresults.tar.gz"

echo "Compressing results"
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "${key}" ${user}@${ip} "cd ~/go/src/github.com/tcrain/cons; tar czf ./${resultsArchive} ./benchresults"

echo "Copying results"
scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "${key}" ${user}@${ip}:"~/go/src/github.com/tcrain/cons/${resultsArchive}" ./

echo "Extracting results"
tar xzf ./${resultsArchive} ./

# rsync -arce "ssh -o StrictHostKeyChecking=no -i ${key}" ${user}@${ip}:"~/go/src/github.com/tcrain/cons/benchresults/" ./benchresults/
