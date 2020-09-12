#!/bin/bash
regions="us-central1-a"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
deleteimage=0 # Delete the generated image at the end of the benchmark
instancetype="n1-highmem-4" # instance type of the nodes
branch="sca-tst"
singleZoneRegion="1"

echo Running binary experiment
#toFolders="./testconfigs/all2all-cbcast-sleep/"
toFolders="./testconfigs/all2all-sleep/ ./testconfigs/all2all-cbcast-sleep/"
nodesPerRegion=5
nodesCount="400 800 1200"

launchNodes=0
shutdownNodes=0
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
