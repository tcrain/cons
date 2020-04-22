regions="us-central1-c us-west1-c us-west2-c us-west3-c"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
deleteimage=0 # Delete the generated image at the end of the benchmark
instancetype="n1-standard-2" # instance type of the nodes
branch="$GITBRANCH"
singleZoneRegion="1"

echo Running binary experiment
toFolders="./testconfigs/s-coin/ ./testconfigs/s-coinpresets/ ./testconfigs/ns-coin/ ./testconfigs/ns-coinpresets/ ./testconfigs/s-coin2/ ./testconfigs/ns-coin2/ ./testconfigs/s-coin2presets/ ./testconfigs/ns-coin2presets/ ./testconfigs/s-coin2echo/ ./testconfigs/ns-coin2echo/ "
# toFolders="./testconfigs/s-coin2echo/"
nodesPerRegion=4
nodesCount="16"

launchNodes=1
shutdownNodes=1
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"

# echo Shutdown
#bash ./scripts/cloudscripts/afterbench.sh "$regions"

# Shutdown the image and the disk
#go run ./cmd/instancesetup/instancesetup.go -sd -dd