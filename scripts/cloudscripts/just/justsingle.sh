#!/bin/bash
set -e
set -o pipefail

homezone=${1:-us-central1-a} # zone from where the benchmarks will be launched
homeinstancetype=${2:-n1-standard-2} # instance type that will launch the benchmarks
user=${3:-$BENCHUSER} # user name to log onto instances
key=${4:-$KEYPATH} # key to use to log onto instances
project=${5:-$PROJECTID} # google cloud project to use
credentialfile=${6:-$OAUTHPATH} # credential file for google cloud
singleZoneCmd="-lrz"

# First check if the instance was already started
if ! inip=$(go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -p "$project" -c "$credentialfile" -i "$homeinstancetype" -z "$homezone" -ii -im cons-image)
then
    # Launch the image that was set up
    inip=$(go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -p "$project" -c "$credentialfile" -i "$homeinstancetype" -z "$homezone" -li -im cons-image)

    # wait for start
    until ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o ConnectTimeout=5 -i "$key" "$user"@"$inip" "exit"; do sleep 5; done
    sleep 25
fi

# Copy the key
scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$credentialfile" "$inip":~/go/src/github.com/tcrain/cons/cloud.json

echo Running rsync with inital instance
bash ./scripts/cloudscripts/rsyncinit.sh "${inip}" "${user}" "${key}"

ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$user"@"$inip"

