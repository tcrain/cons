ip=${1}
profile=${2:-heap}
user=${3:-$BENCHUSER} # user name to log onto instances
key=${4:-$KEYPATH} # key to use to log onto instances

ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -N -i "${key}" -L 6060:localhost:6060 "${user}@${ip}" &
pid=$!
sleep 2

go tool pprof http://localhost:6060/debug/pprof/${profile}

kill $pid