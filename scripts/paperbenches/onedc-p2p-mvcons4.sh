#!/bin/bash
regions="us-central1-a"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
deleteimage=0 # Delete the generated image at the end of the benchmark
instancetype="n1-standard-1" # instance type of the nodes
homeinstancetype="n1-standard-1"
branch="mvcons4"
singleZoneRegion="1"
nodesPerRegion=4

launchNodes=0
shutdownNodes=0

echo Running p2p MvCons4 experiment
toFolders="./testconfigs/p2p-mvcons4-sleep/"
nodesCount="10,4"
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone" "$homeinstancetype"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone" "$homeinstancetype"
