set -e
set -o pipefail
bash ./scripts/setuprpc.sh
sleep 1
for file in ./testconfigs/*.json
do
  echo Running $file
  PRINT_MIN=true ./rpcbench -o $file
done
# ./rpcbench -c 1
# bash ./scripts/killgo.sh
