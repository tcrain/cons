singleZoneCmd=${1:--nlrz}
regions=${2:-us-central1}
project=${3:-$PROJECTID}
credentialPath=${4:-$OAUTHPATH}

# Shut down instances
echo Shutting down instance groups: "$singleZoneCmd" -p "$project" -c "$credentialPath" -r "$regions" -ur
go run cmd/instancesetup/instancesetup.go "$singleZoneCmd" -p "$project" -c "$credentialPath" -r "$regions" -ur

# Remove template
echo Removing instance template: "$singleZoneCmd" -p "$project" -c "$credentialPath" -dit
go run cmd/instancesetup/instancesetup.go "$singleZoneCmd" -p "$project" -c "$credentialPath" -dit
