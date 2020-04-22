set -e
set -o pipefail
bash ./scripts/setuprpc.sh
sleep 1
./rpcbench #-o ./testconfigs/mvbufforward/1.json
# ./rpcbench -c 1
# ./scripts/killgo.sh
