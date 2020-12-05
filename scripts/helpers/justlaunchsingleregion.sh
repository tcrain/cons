launchNodes=${1:-0}

regions="us-central1-a"
homezone="us-central1-a"
genimage=0 # Generate the image for building the benchmark
instancetype="n1-standard-2" # instance type of the nodes
branch="$GITBRANCH"
singleZoneRegion="1"
nodesPerRegion=1
nodesCount="1"
homeinstancetype="n1-standard-2" # instance type that will launch the benchmarks
goversion="1.15" # version of go to use
user=$BENCHUSER # user name to log onto instances
key=$KEYPATH # key to use to log onto instances
project=$PROJECTID # google cloud project to use
credentialfile=$OAUTHPATH # credential file for google cloud
enableprofile=1 # set to 1 to enable profiling

echo Running: bash scripts/cloudscripts/just/justsetup.sh "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$genimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone" "$homeinstancetype" "$goversion" "$user" "$key" "$project" "$credentialfile" "$enableprofile"
bash scripts/cloudscripts/just/justsetup.sh "$regions" "$nodesPerRegion" "$nodesCount" "$launchNodes" "$genimage" "$instancetype" "$branch" "$singleZoneRegion" "$homezone" "$homeinstancetype" "$goversion" "$user" "$key" "$project" "$credentialfile" "$enableprofile"
