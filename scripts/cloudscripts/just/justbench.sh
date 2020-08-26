set -e
set -o pipefail

config=${1}

vars=()

while read -r line
do
    vars+=("$line")
done < .lastjustsetup

inip=${vars[0]}
regions=${vars[2]}
nodesperregion=${vars[3]}
nodecounts=${vars[4]}
launchNodes=${vars[5]}
genimage=${vars[7]}
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

echo Copying "$config" to node "$inip" as ~/go/src/github.com/tcrain/cons/tmptofile.json
scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$config" "$inip":~/go/src/github.com/tcrain/cons/tmptofile.json

# Run the bench
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i "$key" "$user"@"$inip" "
bash --login -c \"
cd ~/go/src/github.com/tcrain/cons/;
echo Running: bash ./scripts/cloudscripts/justcloudbench.sh tmptofile.json ${inip} ${user} ~/.ssh/id_rsa ${project} cloud.json ${singleZoneCmd} ${regions};
bash ./scripts/cloudscripts/justcloudbench.sh tmptofile.json ${inip} ${user} ~/.ssh/id_rsa ${project} cloud.json ${singleZoneCmd} ${regions}\""

