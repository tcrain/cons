regions="us-central1 us-east4 us-west1 europe-north1 europe-west2 europe-west3 asia-east1 australia-southeast1"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
deleteimage=0 # Delete the generated image at the end of the benchmark
instancetype="n1-standard-2" # instance type of the nodes
branch="$GITBRANCH"
singleZoneRegion="0"

echo Running binary experiment
# toFolders="./testconfigs/s-coinscalepresets/ ./testconfigs/ns-coinscalepresets/"
toFolders="./testconfigs/ns-coinscalepresetsonce/ ./testconfigs/s-coinscalepresets/"
nodesPerRegion=6
nodesCount="8 16 32 48"
#nodesCount="48"

launchNodes=1
shutdownNodes=1
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"

# echo Shutdown
#bash ./scripts/cloudscripts/afterbench.sh "$regions"

# Shutdown the image and the disk
#go run ./cmd/instancesetup/instancesetup.go -sd -dd