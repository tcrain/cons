set -e
set -o pipefail

ip=$1
homezone=${2:-us-central1-a} # zone from where the benchmarks will be launched
user=${3:-$BENCHUSER} # user name to log onto instances
key=${4:-$KEYPATH} # key to use to log onto instances
project=${5:-$PROJECTID} # google cloud project to use
credentialfile=${6:-$OAUTHPATH} # credential file for google cloud

inip=$(go run ./cmd/instancesetup/instancesetup.go -p "$project" -c "$credentialfile" -z "$homezone" -ii -im cons-image)

echo Syncing files
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i $key $user@$inip "
  rsync -arce \"ssh -o StrictHostKeyChecking=no \" ~/go/bin/dlv ${user}@${ip}:\"~/dlv\";
  rsync -arce \"ssh -o StrictHostKeyChecking=no \" --include=\"*/\" --include=\"*.go\" --exclude=\"*\" ~/go/src/github.com/tcrain/cons/ ${user}@${ip}:\"~/go/src/github.com/tcrain/cons/\";
"

echo Connecting to node $ip
ssh -t -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i ${key} ${user}@${ip} "~/dlv attach \$(pidof rpcnode) ~/go/src/github.com/tcrain/cons/rpcnode"
