set -e
set -o pipefail

user=$1
ip=$2
key=$3

# Update
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i $key $user@$ip "
sudo apt-get -y update
sudo apt-get -y upgrade"

# reboot
echo "Rebooting"
set +e
ssh -i $key -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o ServerAliveInterval=1 -o ServerAliveCountMax=1 $user@$ip "
sudo reboot"
set -e

echo "Waiting for restart"
# wait for reboot
until ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o ConnectTimeout=5 -i $key $user@$ip "exit"; do sleep 5; done
sleep 15
