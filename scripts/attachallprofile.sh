#!/bin/bash
ip=${1}
user=${3:-$BENCHUSER} # user name to log onto instances
key=${4:-$KEYPATH} # key to use to log onto instances

sshCmd="htop"
if [ "$ip" != "" ]
then
  sshCmd="ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -i ${key} ${user}@${ip} htop"
  ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -N -i "${key}" -L 6060:localhost:6060 "${user}@${ip}" &
  pid=$!
  sleep 2
fi

cmd1="go tool pprof http://localhost:6060/debug/pprof/profile"
cmd2="go tool pprof http://localhost:6060/debug/pprof/heap"
cmd3="go tool pprof http://localhost:6060/debug/pprof/allocs"

bash ./scripts/tmuxcmd.sh profile "$sshCmd" "$cmd1" "$cmd2" "$cmd3"

if [ "$ip" != "" ]
then
  kill $pid
fi