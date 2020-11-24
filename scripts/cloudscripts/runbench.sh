set -e
set -o pipefail

pregip=${1}
tofolders=${2}
singleZoneCmd=${3:--nlrz}
regions=${4:-us-central1}
nodecounts=${5:-4}
user=${6:-$BENCHUSER}
key=${7:-$KEYPATH}
project=${8:-$PROJECTID}
credentialPath=${9:-$OAUTHPATH}
ipfile=${10:-./benchIPfile}
pregport=${11:-4534}
nwtest=${12:-1}
enableprofile=${13:-0}

benchid=$(date +"%m-%d-%y_%T")

read -ra toarray <<< "$tofolders"

echo "$pregip
${toarray[0]}
$benchid
$nodecounts
$user
$ipfile
$key
$pregport
$nwtest
$project
$credentialPath
$singleZoneCmd
$regions
$tofolders
$enableprofile" > .lastrun

bash ./scripts/cloudscripts/retrybench.sh
