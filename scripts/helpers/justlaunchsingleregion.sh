regions="us-central1-a"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
instancetype="n1-standard-2" # instance type of the nodes
branch="$GITBRANCH"
singleZoneRegion="1"
nodesPerRegion=4
nodesCount="4"
launchNodes=1

echo Running: bash scripts/cloudscripts/just/justsetup.sh "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$genimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/just/justsetup.sh "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$genimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
