set -e
set -o pipefail

config=${1}
pregip=${2}
user=${3}
key=${4}
project=${5}
credentialPath=${6}
singleZoneCmd=${7}
regions=${8}

logname="./benchresults/run-$config.log"
# Get ips
bash ./scripts/cloudscripts/getips.sh "$singleZoneCmd" "$regions" "$project" "$credentialPath"

echo Running setup
bash ./scripts/setupnodes.sh "$user" benchIPfile "$key" "$pregip" 4534

echo
echo "Running: ./rpcbench -o $config -f benchIPfile -p $pregip" | tee -a "$logname"
tee -a "$logname" < "$config"
sleep 2
./rpcbench -o "$config" -f benchIPfile -p "$pregip" 2>&1 | tee -a "$logname"
