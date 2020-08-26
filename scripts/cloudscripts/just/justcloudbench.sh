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
doInitialSetup=${9}

logname="./benchresults/run-$config.log"

if [ "$doInitialSetup" -eq 1 ]
then
    echo Doing intial node setup
    # Get ips
    bash ./scripts/cloudscripts/getips.sh "$singleZoneCmd" "$regions" "$project" "$credentialPath"

    echo Running setup
    bash ./scripts/setupnodes.sh "$user" benchIPfile "$key" "$pregip" 4534

fi

echo
echo "Running: ./rpcbench -o $config -f benchIPfile -p $pregip:4534" | tee -a "$logname"
tee -a "$logname" < "$config"
sleep 2
./rpcbench -o "$config" -f benchIPfile -p "$pregip:4534" 2>&1 | tee -a "$logname"
