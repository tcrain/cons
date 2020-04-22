set -e
set -o pipefail

singleZoneCmd=${1:--nlrz}
regions=${2:-us-central1}
project=${3:-$PROJECTID}
credentialPath=${4:-$OAUTHPATH}

# Get ips
echo Getting IPs: "$singleZoneCmd" -p "$project" -c "$credentialPath" -r "$regions" -gi
go run cmd/instancesetup/instancesetup.go "$singleZoneCmd"  -p "$project" -c "$credentialPath" -r "$regions" -gi > benchIPfile
