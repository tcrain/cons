set -e
set -o pipefail

pregIP=$1
nodes=${2:-1}
singleZoneCmd=${3:--nlrz}
regions=${4:-us-central1}
instancetype=${5:-n1-standard-2}
user=${6:-$BENCHUSER}
key=${7:-$KEYPATH}
project=${8:-$PROJECTID}
credentialPath=${9:-$OAUTHPATH}
launchNodes=${10:-1}

if [ "$launchNodes" -eq 1 ]
then

  # Create instance template
  echo Creating instance template: "$singleZoneCmd" -p "$project" -c "$credentialPath" -i "$instancetype" -cit
  go run cmd/instancesetup/instancesetup.go "$singleZoneCmd" -p "$project" -c "$credentialPath" -i "$instancetype" -cit

  # launch nodes
  echo Launching nodes: "$singleZoneCmd" -p "$project" -c "$credentialPath" -r "$regions" -n "$nodes" -lr
  go run cmd/instancesetup/instancesetup.go "$singleZoneCmd" -p "$project" -c "$credentialPath" -r "$regions" -n "$nodes" -lr

fi

# Get ips
bash ./scripts/cloudscripts/getips.sh "$singleZoneCmd" "$regions" "$project" "$credentialPath"

setuptimeout="120s"

# Setup basic nodes
echo Setting up basic nodes
if ! timeout "$setuptimeout" bash ./scripts/setupbasicimage.sh "$user" benchIPfile "$pregIP" "$key"
then

  for i in {1..5}
  do
    echo Error doing initial basic node setup, will retry after 30 seconds
    sleep 30s

    # get ips
    if ! bash ./scripts/cloudscripts/getips.sh "$singleZoneCmd" "$regions" "$project" "$credentialPath"
    then
      continue
    fi

    if timeout "$setuptimeout" bash ./scripts/setupbasicimage.sh "$user" benchIPfile "$pregIP" "$key"
    then
      break
    fi

    if [ $i -eq 5 ]
    then
      echo Failed benchmark too many times, exiting
      exit 1
    fi
  done
fi