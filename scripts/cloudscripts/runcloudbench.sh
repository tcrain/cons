set -e
set -o pipefail

pregip=${1}
tofolders=${2}
singleZoneCmd=${3:--nlrz}
regions=${4:-us-central1}
nodesperregion=${5:-1}
nodecounts=${6:-4}
instancetype=${7:-n1-standard-2}
user=${8:-$BENCHUSER}
key=${9:-$KEYPATH}
project=${10:-$PROJECTID}
crednetialfile=${11:-$OAUTHPATH}
launchNodes=${12:-1}
shutdownNodes=${13:-1}
enableprofile=${14:-$PROF}

# Build
echo "Building"
bash ./scripts/benchscripts/profilesetup.sh "$enableprofile"
bash ./scripts/buildgo.sh 1

# Launch the nodes
echo Running: bash ./scripts/cloudscripts/prepareinstances.sh "$pregip" "$nodesperregion" "$singleZoneCmd" "$regions" "$instancetype" "$user" "$key" "$project" "$crednetialfile" "$launchNodes"
bash ./scripts/cloudscripts/prepareinstances.sh "$pregip" "$nodesperregion" "$singleZoneCmd" "$regions" "$instancetype" "$user" "$key" "$project" "$crednetialfile" "$launchNodes"

# Make test options
go run ./cmd/gento/gento.go

# Run the bench
echo Running: bash ./scripts/cloudscripts/runbench.sh "$pregip" "$tofolders" "$singleZoneCmd" "$regions" "$nodecounts" "$user" "$key" "$project" "$crednetialfile"
bash ./scripts/cloudscripts/runbench.sh "$pregip" "$tofolders" "$singleZoneCmd" "$regions" "$nodecounts" "$user" "$key" "$project" "$crednetialfile"

if [ "$shutdownNodes" -eq 1 ]
then
  # Shut down the bench nodes
  bash ./scripts/cloudscripts/afterbench.sh "$singleZoneCmd" "$regions" "$project" "$crednetialfile"
else
  echo Not shutting down nodes, run command to shut down:
  echo bash ./scripts/cloudscripts/afterbench.sh "$singleZoneCmd" "$regions" "$project" "$crednetialfile"
fi
#
#
## copy json file
#scp -i $KEYPATH $OAUTHPATH $inip:~/go/src/github.com/$BENCHUSER/cons
#
## Build
#bash ./scripts/buildgo.sh
#
## Launch the nodes
#bash ./scripts/cloudscripts/prepareinstances.sh "$pregip" "$nodes" "$regions"
#
## Make test options
#go run ./cmd/gento/gento.go
#
## Run the bench
#bash ./scripts/cloudscripts/runbench.sh "$pregip" "$tofolder" "$regions" "$nodecounts"
#
## bash ./scripts/Bench.sh "$pregip" "$tofolder" "$nodes" "$user" "$ipfile" "$key" 4534 1
#
## Shut down the bench nodes
#bash ./scripts/cloudscripts/afterbench.sh "$regions"
#
## Shutdown the image and the disk
#go run ./cmd/instancesetup/instancesetup.go -sd -dd
#
## Delete the image
#go run ./cmd/instancesetup/instancesetup.go -im cons-image -us