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


ips=$(go run cmd/instancesetup/instancesetup.go "$singleZoneCmd"  -p "$project" -c "$credentialfile" -r "$regions" -gi -eip)

commands=()
for ip in $ips
do
  ip=${ip%:*}
  commands+=("bash ./scripts/cloudscripts/attachlog.sh $ip $user $key")
done

bash ./scripts/tmuxwindow.sh alllogs "${commands[@]}"