set -e
set -o pipefail

pregip=${1}
tofolders=${2}
singleZoneCmd=${3:--nlrz}
regions=${4:-us-central1}
nodesperregion=${5:-1}
nodecounts=${6:-4}
constypes=${7:-2}
instancetype=${8:-n1-standard-2}
user=${9:-$BENCHUSER}
key=${10:-$KEYPATH}
project=${11:-$PROJECTID}
crednetialfile=${12:-$OAUTHPATH}
launchNodes=${12:-1}
shutdownNodes=${13:-1}

# Build
bash ./scripts/buildgo.sh 1

# Launch the nodes
echo Running: bash ./scripts/cloudscripts/prepareinstances.sh "$pregip" "$nodesperregion" "$singleZoneCmd" "$regions" "$instancetype" "$user" "$key" "$project" "$crednetialfile" 0
bash ./scripts/cloudscripts/prepareinstances.sh "$pregip" "$nodesperregion" "$singleZoneCmd" "$regions" "$instancetype" "$user" "$key" "$project" "$crednetialfile" 0

# Make test options
go run ./cmd/gento/gento.go

# Run the bench
echo Running: bash ./scripts/cloudscripts/retrybench.sh
bash ./scripts/cloudscripts/retrybench.sh

if [ "$shutdownNodes" -eq 1 ]
then
  # Shut down the bench nodes
  bash ./scripts/cloudscripts/afterbench.sh "$singleZoneCmd" "$regions" "$project" "$crednetialfile"
else
  echo Not shutting down nodes, run command to shut down:
  echo bash ./scripts/cloudscripts/afterbench.sh "$singleZoneCmd" "$regions" "$project" "$crednetialfile"
fi
