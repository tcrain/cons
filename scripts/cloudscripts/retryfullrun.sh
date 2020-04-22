set -e
set -o pipefail

vars=()

while read -r line
do
    vars+=("$line")
done < .fulllastrun

inip=${vars[0]}
tofolders=${vars[1]}
regions=${vars[2]}
nodesperregion=${vars[3]}
nodecounts=${vars[4]}
launchNodes=${vars[5]}
shutdownNodes=${vars[6]}
genimage=${vars[7]}
deleteimage=${vars[8]}
constypes=${vars[9]}
instancetype=${vars[10]}
branch=${vars[11]}
homezone=${vars[12]}
homeinstancetype=${vars[13]}
goversion=${vars[14]}
user=${vars[15]}
key=${vars[16]}
project=${vars[17]}
credentialfile=${vars[18]}
singleZoneCmd=${vars[19]}

# Format input
printf -v inip %q "$inip"
printf -v tofolders %q "$tofolders"
printf -v regions %q "${regions}"
printf -v nodesperregion %q "${nodesperregion}"
printf -v nodecounts %q "${nodecounts}"
printf -v constypes %q "${constypes}"
printf -v instancetype %q "${instancetype}"
printf -v user %q "${user}"
printf -v key %q "${key}"
printf -v project %q "${project}"
printf -v credentialfile %q "${credentialfile}"

# Run the bench
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$user"@"$inip" "
bash --login -c \"
cd ~/go/src/github.com/tcrain/cons/;
git pull;
echo Running: bash ./scripts/cloudscripts/retryruncloudbench.sh ${inip} ${tofolders} ${singleZoneCmd} ${regions} ${nodesperregion} ${nodecounts} ${constypes} ${instancetype} ${user} ~/.ssh/id_rsa ${project} cloud.json ${launchNodes} ${shutdownNodes};
bash ./scripts/cloudscripts/retryruncloudbench.sh ${inip} ${tofolders} ${singleZoneCmd} ${regions} ${nodesperregion} ${nodecounts} ${constypes} ${instancetype} ${user} ~/.ssh/id_rsa ${project} cloud.json ${launchNodes} ${shutdownNodes}\""

# Get the results
bash ./scripts/graphscripts/syncresults.sh "$inip" "$user" "$key"

if [ "$shutdownNodes" -eq 1 ]
then
  # Shutdown the image and the disk
  go run ./cmd/instancesetup/instancesetup.go "$singleZoneCmd" -p "$project" -c "$credentialfile" -z "$homezone" -sd -dd
fi

if [ "$deleteimage" -eq 1 ]
then
  # Delete the image
  go run ./cmd/instancesetup/instancesetup.go "$singleZoneCmd" -im cons-image -p "$project" -c "$credentialfile" -z "$homezone" -us
fi