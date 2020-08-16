#!/bin/bash
set -e
set -o pipefail

if [ "$#" -ne 5 ]
then
    echo "This command syncs and starts processes on remote nodes necessary for running a benchmark"
    echo "Usage: \"./scripts/setupnodes.sh user ipfile ssh-key participant-register-ip participant-register-port {DoRsync (0 for no, 1 for yes, default 1)}\""
    exit
fi

user=$1
ipfile=$2
key=$3
pregip=$4
pregport=$5
#dorsync=$6

# echo "Building"
# bash ./scripts/buildgo.sh
echo "Restarting nodes"
./runcmd -f "$ipfile" -k "$key" -u "$user" nohup "sudo bash --login -c \"( sleep 5; reboot ) > \dev\null 2>&1 &\""
echo "Done restart, sleeping 20 seconds"
sleep 20

echo "Killing procs"
bash ./scripts/killgo.sh
./runcmd -f "$ipfile" -k "$key" -u "$user" bash ~/go/src/github.com/tcrain/cons/scripts/killgo.sh
ssh -o "StrictHostKeyChecking no" -i "${key}" "${user}"@"${pregip}" "bash ~/go/src/github.com/tcrain/cons/scripts/killgo.sh"

#if [ "$dorsync" -eq 1 ]
#then
#    echo "Calling rsync"
#    ./runcmd -f "$ipfile" -k "$key" -u "$user" -r ~/go/src/github.com/tcrain/cons/ "~/go/src/github.com/tcrain/cons/"
#    echo "Done rsync"
#fi
pregoutfile=$(date +"%m-%d-%y_%T")"preg.out"
rpcoutfile=$(date +"%m-%d-%y_%T")"preg.out"

# start the participant register
echo "Starting the participant register at $pregip"
ssh -o "StrictHostKeyChecking no" -i "${key}" "${user}"@"${pregip}" "bash --login -c \"nohup ~/go/src/github.com/tcrain/cons/preg -p ${pregport} > ${pregoutfile} 2>&1 &\""

echo "Starting rpc nodes" # ++ is a special symbol that means put the node ip there
./runcmd -k "${key}" -u "${user}" -f "${ipfile}" nohup "bash --login -c \"~/go/src/github.com/tcrain/cons/rpcnode" -i ++  "> ${rpcoutfile} 2>&1 &\"" # start the rpc servers at each of the nodes
