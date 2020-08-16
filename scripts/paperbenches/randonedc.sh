regions="us-central1-a"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
deleteimage=0 # Delete the generated image at the end of the benchmark
instancetype="n1-standard-2" # instance type of the nodes
branch="$GITBRANCH"
singleZoneRegion="1"

echo Running binary experiment
toFolders="./testconfigs/simplebin/"
# toFolders="./testconfigs/s-coinpresets/ ./testconfigs/ns-coinpresets/ ./testconfigs/s-coin/ ./testconfigs/ns-coin/ ./testconfigs/s-coin2/ ./testconfigs/ns-coin2/ ./testconfigs/s-coin2presets/ ./testconfigs/ns-coin2presets/ ./testconfigs/s-coin2echo/ ./testconfigs/ns-coin2echo/"
# toFolders="./testconfigs/s-coin2echo/"
# toFolders="./testconfigs/s-coin/ ./testconfigs/s-coinpresets/"
nodesPerRegion=4
nodesCount="4"

launchNodes=1
shutdownNodes=1
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"

# echo Shutdown
#bash ./scripts/cloudscripts/afterbench.sh "$regions"

# Shutdown the image and the disk
#go run ./cmd/instancesetup/instancesetup.go -sd -dd