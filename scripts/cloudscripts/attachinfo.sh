#!/bin/bash

ip=$1
user=${2:-$BENCHUSER} # user name to log onto instances
key=${3:-$KEYPATH} # key to use to log onto instances

if [ "$ip" == "" ]
then
  bash ./scripts/tmuxcmd.sh info htop "sudo iotop -o" "sudo iftop" "tail -F rpcnode.out"
else
  htop="ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -i ${key} ${user}@${ip} \"sudo htop\""
  iotop="ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -i ${key} ${user}@${ip} \"sudo iotop -o\""
  iftop="ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -i ${key} ${user}@${ip} \"sudo iftop\""
  bash ./scripts/tmuxcmd.sh info "$htop" "$iotop" "$iftop" "bash ./scripts/cloudscripts/attachlog.sh $ip $user $key"
fi