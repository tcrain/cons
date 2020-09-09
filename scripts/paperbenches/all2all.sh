#regions="us-central1 us-east4 europe-north1 europe-west2"
regions="us-east4 us-west1 europe-north1 europe-west2 australia-southeast1"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
deleteimage=0 # Delete the generated image at the end of the benchmark
instancetype="n1-standard-4" # instance type of the nodes
branch="sca-tst"
singleZoneRegion="0"

echo Running all to all
nodesPerRegion=5

launchNodes=1
shutdownNodes=0

toFolders="./testconfigs/all2all/"
nodesCount="5, 10, 15, 20, 25"
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"

launchNodes=0
toFolders="./testconfigs/all2all-sleep/"
nodesCount="50, 100, 150, 200, 250"
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"

shutdownNodes=1
toFolders="./testconfigs/all2all-sleep/"
nodesCount="400, 800, 1200, 1600, 2000"
echo bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
bash scripts/cloudscripts/fullrun.sh "$toFolders" "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$shutdownNodes" "$genimage" "$deleteimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone"
