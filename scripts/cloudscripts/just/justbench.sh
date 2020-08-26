set -e
set -o pipefail

config=${1}
doInitialSetup=${2:-1}

vars=()

while read -r line
do
    vars+=("$line")
done < .lastjustsetup

inip=${vars[0]}
regions=${vars[1]}
nodesperregion=${vars[2]}
nodecounts=${vars[3]}
launchNodes=${vars[4]}
genimage=${vars[5]}
instancetype=${vars[6]}
branch=${vars[7]}
homezone=${vars[8]}
homeinstancetype=${vars[9]}
goversion=${vars[10]}
user=${vars[11]}
key=${vars[12]}
project=${vars[13]}
credentialfile=${vars[14]}
singleZoneCmd=${vars[15]}

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

echo Copying "$config" to node "$inip" as ~/go/src/github.com/tcrain/cons/tmptofile.json
scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$config" "$inip":~/go/src/github.com/tcrain/cons/tmptofile.json

# Run the bench
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$user"@"$inip" "
bash --login -c \"
cd ~/go/src/github.com/tcrain/cons/;
echo Running: bash ./scripts/cloudscripts/just/justcloudbench.sh tmptofile.json ${inip} ${user} ~/.ssh/id_rsa ${project} cloud.json ${singleZoneCmd} ${regions} ${doInitialSetup};
bash ./scripts/cloudscripts/just/justcloudbench.sh tmptofile.json ${inip} ${user} ~/.ssh/id_rsa ${project} cloud.json ${singleZoneCmd} ${regions} ${doInitialSetup}\""

