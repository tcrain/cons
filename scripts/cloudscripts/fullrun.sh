set -e
set -o pipefail

tofolders=${1} # benchmarks to run
regions=${2:-us-central1} # regions to run nodes in
nodesperregion=${3:-1} # number of nodes to launch in each region
nodecounts=${4:-4} # list of node counts to run each benchmark
launchNodes=${5:-1} # Launch bench nodes
shutdownNodes=${6:-1} # Shutdown bench nodes
genimage=${7:-0} # Generate the image for building the benchmark
deleteimage=${8:-0} # Delete the generated image at the end of the benchmark
instancetype=${9:-n1-standard-1} # instance type of the nodes
branch=${10:-$GITBRANCH} # git branch to use
singleZoneRegion=${11:-0} # run nodes in the same region in the same zone
homezone=${12:-us-central1-a} # zone from where the benchmarks will be launched
homeinstancetype=${13:-n1-standard-2} # instance type that will launch the benchmarks
goversion=${14:-1.14.2} # version of go to use
user=${15:-$BENCHUSER} # user name to log onto instances
key=${16:-$KEYPATH} # key to use to log onto instances
project=${17:-$PROJECTID} # google cloud project to use
credentialfile=${18:-$OAUTHPATH} # credential file for google cloud

# regions="europe-north1 europe-west3 us-central1 us-west1"

if [ "$singleZoneRegion" -eq 1 ]
then
  singleZoneCmd="-lrz"
else
  singleZoneCmd="-nlrz"
fi

if [ "$genimage" -eq 1 ]
then
  # make the image
  echo Making image
  bash ./scripts/cloudscripts/makeimage.sh "$homeinstancetype" "$branch" "$homezone" "$goversion" "$user" "$key" "$project" "$credentialfile"
fi

if [ "$launchNodes" -eq 1 ]
then
  # Launch the image that was set up
  inip=$(go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -p "$project" -c "$credentialfile" -i "$homeinstancetype" -z "$homezone" -li -im cons-image)
else
  # The instance should already be started, just get the ip
  inip=$(go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -p "$project" -c "$credentialfile" -i "$homeinstancetype" -z "$homezone" -ii -im cons-image)
fi

# wait for start
until ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o ConnectTimeout=5 -i "$key" "$user"@"$inip" "exit"; do sleep 5; done

# Copy the key
scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$credentialfile" "$inip":~/go/src/github.com/tcrain/cons/cloud.json

echo "$inip
$tofolders
$regions
$nodesperregion
$nodecounts
$launchNodes
$shutdownNodes
$genimage
$deleteimage
$instancetype
$branch
$singleZoneRegion
$homezone
$homeinstancetype
$goversion
$user
$key
$project
$credentialfile
$singleZoneCmd" > .fulllastrun

# Format input
printf -v inip %q "$inip"
printf -v tofolders %q "$tofolders"
printf -v singleZoneCmd %q "${singleZoneCmd}"
printf -v regions %q "${regions}"
printf -v nodesperregion %q "${nodesperregion}"
printf -v nodecounts %q "${nodecounts}"
printf -v instancetype %q "${instancetype}"
printf -v user %q "${user}"
printf -v key %q "${key}"
printf -v project %q "${project}"
printf -v credentialfile %q "${credentialfile}"

# Run the bench
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$user"@"$inip" "
bash --login -c \"
cd ~/go/src/github.com/tcrain/cons/;
git pull;
git checkout ${branch};
git pull;
echo Running: bash ./scripts/cloudscripts/runcloudbench.sh ${inip} ${tofolders} ${singleZoneCmd} ${regions} ${nodesperregion} ${nodecounts} ${instancetype} ${user} ~/.ssh/id_rsa ${project} cloud.json ${launchNodes} ${shutdownNodes};
bash ./scripts/cloudscripts/runcloudbench.sh ${inip} ${tofolders} ${singleZoneCmd} ${regions} ${nodesperregion} ${nodecounts} ${instancetype} ${user} ~/.ssh/id_rsa ${project} cloud.json ${launchNodes} ${shutdownNodes}\""

# Get the results
bash ./scripts/graphscripts/syncresults.sh "$inip" "$user" "$key"

if [ "$shutdownNodes" -eq 1 ]
then
  # Shutdown the image and the disk
  echo Shutting down launch node
  go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -p "$project" -c "$credentialfile" -z "$homezone" -sd -dd
else
  echo Launch node not shut down, run command to shut down launch node:
  echo go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -p "$project" -c "$credentialfile" -z "$homezone" -sd -dd
fi

if [ "$deleteimage" -eq 1 ]
then
  # Delete the image
  go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -im cons-image -p "$project" -c "$credentialfile" -z "$homezone" -us
else
  echo Not deleting the test image, run command to delete image:
  echo go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -im cons-image -p "$project" -c "$credentialfile" -z "$homezone" -us
fi
