
singleZoneCmd=${1:--nlrz}
project=${2:-$PROJECTID}
credentialPath=${3:-$OAUTHPATH}

regions=

IPs=$(go run cmd/instancesetup/instancesetup.go "$singleZoneCmd"  -p "$project" -c "$credentialPath" -r "$regions" -gi > benchIPfile)

echo "IPs are $IPs"