set -e
set -o pipefail

instancetype=${1:-n1-standard-2}
branch=${2:-$GITBRANCH}
zone=${3:-us-central1-a}
goversion=${4:-1.15}
user=${5:-$BENCHUSER}
key=${6:-$KEYPATH}
project=${7:-$PROJECTID}
credentialPath=${8:-$OAUTHPATH}
imageName=${9:-cons-image}

echo "Launching an image: -p $project -c $credentialPath -i $instancetype -z $zone -li"
ip=$(go run cmd/instancesetup/instancesetup.go -p "$project" -c "$credentialPath" -i "$instancetype" -z "$zone" -li)

echo "Running setup script: ./scripts/imagesetuplocal.sh $user $ip $key $goversion $branch 0"
bash ./scripts/imagesetuplocal.sh "$user" "$ip" "$key" "$goversion" "$branch" 0

echo "Shuting instance down to create image"
go run cmd/instancesetup/instancesetup.go  -p "$project" -c "$credentialPath" -z "$zone" -sd

echo "Creating image: -p $project -c $credentialPath -z $zone -s"
go run cmd/instancesetup/instancesetup.go  -p "$project" -c "$credentialPath" -z "$zone" -im "$imageName" -s

# echo "Shuting down instance: -p $project -c $credentialPath -z $zone -s"
# go run cmd/instancesetup/instancesetup.go  -p "$project" -c "$credentialPath" -z "$zone" -sd
