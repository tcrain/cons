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


# Shut down the bench nodes
bash ./scripts/cloudscripts/afterbench.sh "$singleZoneCmd" "$regions" "$project" "$credentialfile"

echo Shutting down launch node
go run ./cmd/instancesetup/instancesetup.go $singleZoneCmd -p "$project" -c "$credentialfile" -z "$homezone" -sd -dd
