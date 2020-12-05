#!/bin/bash
#regions="us-central1 us-east4 europe-north1 europe-west2"
regions="us-central1 us-east4 us-west1 europe-north1 europe-west2 europe-west3 asia-east1 australia-southeast1"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
deleteimage=0 # Delete the generated image at the end of the benchmark
instancetype="n1-standard-2" # instance type of the nodes
branch="$GITBRANCH"
singleZoneRegion="0"

echo Running binary experiment
toFolders="./testconfigs/s-coin/" # ./testconfigs/ns-coin/ ./testconfigs/s-coinpresets/ ./testconfigs/ns-coinpresets/ ./testconfigs/s-coincombine/" # ./testconfigs/s-coinbyzpresets/ ./testconfigs/ns-coinbyzpresets/
# toFolders="./testconfigs/s-coin/ ./testconfigs/s-coinpresets/ ./testconfigs/s-coinbyz/ ./testconfigs/s-coinbyzpresets/ ./testconfigs/ns-coinbyzpresets/ ./testconfigs/s-coincombine/"
# toFolders="./testconfigs/ns-coinbyzpresets/"
# toFolders="./testconfigs/tmps-coinbyzpresets/"
nodesPerRegion=2
nodesCount="16"

launchNodes=1
shutdownNodes=1
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"

# echo Shutdown
#bash ./scripts/cloudscripts/afterbench.sh "$regions"

# Shutdown the image and the disk
#go run ./cmd/instancesetup/instancesetup.go -sd -dd