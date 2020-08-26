set -e
set -o pipefail

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


# Shut down the bench nodes
bash ./scripts/cloudscripts/afterbench.sh "$singleZoneCmd" "$regions" "$project" "$credentialfile"
