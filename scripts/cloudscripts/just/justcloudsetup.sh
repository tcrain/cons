set -e
set -o pipefail

pregip=${1}
singleZoneCmd=${2:--nlrz}
regions=${3:-us-central1}
nodesperregion=${4:-1}
instancetype=${5:-n1-standard-2}
user=${6:-$BENCHUSER}
key=${7:-$KEYPATH}
project=${8:-$PROJECTID}
crednetialfile=${9:-$OAUTHPATH}
launchNodes=${10:-1}
enableprofile=${11:-$PROF}

# Build
bash ./scripts/benchscripts/profilesetup.sh "$enableprofile"
bash ./scripts/buildgo.sh 1

if [ "$launchNodes" -eq 1 ]
then
    # Launch the nodes
    echo Running: bash ./scripts/cloudscripts/prepareinstances.sh "$pregip" "$nodesperregion" "$singleZoneCmd" "$regions" "$instancetype" "$user" "$key" "$project" "$crednetialfile" "$launchNodes"
    bash ./scripts/cloudscripts/prepareinstances.sh "$pregip" "$nodesperregion" "$singleZoneCmd" "$regions" "$instancetype" "$user" "$key" "$project" "$crednetialfile" "$launchNodes"
fi

# Make test options
go run ./cmd/gento/gento.go

echo "Calling rsync"
./runcmd -f benchIPfile -k "$key" -u "$user" -r ~/go/src/github.com/tcrain/cons/rpcnode "~/go/src/github.com/tcrain/cons/rpcnode"
./runcmd -f benchIPfile -k "$key" -u "$user" -r ~/go/src/github.com/tcrain/cons/scripts/ "~/go/src/github.com/tcrain/cons/scripts/"
echo "Done rsync"

bash ./scripts/setupnodes.sh "$user" benchIPfile "$key" "$pregip" 4534

sleep 3

