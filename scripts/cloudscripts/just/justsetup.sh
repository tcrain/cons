set -e
set -o pipefail

regions=${1:-us-central1} # regions to run nodes in
nodesperregion=${2:-1} # number of nodes to launch in each region
nodecounts=${3:-4} # list of node counts to run each benchmark
launchNodes=${4:-1} # Launch bench nodes
genimage=${5:-0} # Generate the image for building the benchmark
instancetype=${6:-n1-standard-1} # instance type of the nodes
branch=${7:-$GITBRANCH} # git branch to use
singleZoneRegion=${8:-0} # run nodes in the same region in the same zone
homezone=${9:-us-central1-a} # zone from where the benchmarks will be launched
homeinstancetype=${10:-n1-standard-2} # instance type that will launch the benchmarks
goversion=${11:-1.15} # version of go to use
user=${12:-$BENCHUSER} # user name to log onto instances
key=${13:-$KEYPATH} # key to use to log onto instances
project=${14:-$PROJECTID} # google cloud project to use
credentialfile=${15:-$OAUTHPATH} # credential file for google cloud

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
sleep 15

# Copy the key
scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$credentialfile" "$inip":~/go/src/github.com/tcrain/cons/cloud.json

# Format input
printf -v inip %q "$inip"
printf -v singleZoneCmd %q "${singleZoneCmd}"
printf -v regions %q "${regions}"
printf -v nodesperregion %q "${nodesperregion}"
printf -v nodecounts %q "${nodecounts}"
printf -v instancetype %q "${instancetype}"
printf -v user %q "${user}"
printf -v key %q "${key}"
printf -v project %q "${project}"
printf -v credentialfile %q "${credentialfile}"

echo "$inip
$regions
$nodesperregion
$nodecounts
$launchNodes
$genimage
$instancetype
$branch
$homezone
$homeinstancetype
$goversion
$user
$key
$project
$credentialfile
$singleZoneCmd" > .lastjustsetup


# Run the setup
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$user"@"$inip" "
bash --login -c \"
cd ~/go/src/github.com/tcrain/cons/;
git pull;
git checkout ${branch};
git pull;
echo Running: bash ./scripts/cloudscripts/just/justcloudsetup.sh ${inip} ${singleZoneCmd} ${regions} ${nodesperregion} ${instancetype} ${user} ~/.ssh/id_rsa ${project} cloud.json;
bash ./scripts/cloudscripts/just/justcloudsetup.sh ${inip} ${singleZoneCmd} ${regions} ${nodesperregion} ${instancetype} ${user} ~/.ssh/id_rsa ${project} cloud.json\""


