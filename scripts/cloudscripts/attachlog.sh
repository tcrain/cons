set -e
set -o pipefail

ip=$1
user=${2:-$BENCHUSER} # user name to log onto instances
key=${3:-$KEYPATH} # key to use to log onto instances

echo Connecting to node $ip
ssh -t -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i ${key} ${user}@${ip} "tail -F -n 100 ~/rpcnode.out"
