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

    echo "Calling rsync"
    ./runcmd -f benchIPfile -k "$key" -u "$user" -r ~/go/src/github.com/tcrain/cons/rpcnode "~/go/src/github.com/tcrain/cons/rpcnode"
    ./runcmd -f benchIPfile -k "$key" -u "$user" -r ~/go/src/github.com/tcrain/cons/scripts/ "~/go/src/github.com/tcrain/cons/scripts/"
    echo "Done rsync"

    echo Running setup
    bash ./scripts/setupnodes.sh "$user" benchIPfile "$key" "$pregip" 4534

fi

sleep 2
echo "Nodes started, press any key to start benchmark"
read -n 1 -s -r

echo
echo "Running: ./rpcbench -o $config -f benchIPfile -p $pregip:4534" | tee -a "$logname"
tee -a "$logname" < "$config"
sleep 2
./rpcbench -o "$config" -f benchIPfile -p "$pregip:4534" 2>&1 | tee -a "$logname"
