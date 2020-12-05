set -e
set -o pipefail

user=$1
ipfile=$2
pregip=$3
key=$4

echo "Setting up enviroment variable"
./runcmd -k "${key}" -u "${user}" -f "${ipfile}" "echo '
export LD_LIBRARY_PATH=\$LD_LIBRARY_PATH:~/liboqs/build/lib/
' >> ~/.profile;
mkdir -p ~/liboqs/build/lib/
mkdir -p ~/go/src/github.com/tcrain/cons/scripts;
sudo apt-get -y install rsync htop iftop iotop"

echo "Calling rsync on liboqs"
./runcmd -f "$ipfile" -k "$key" -u "$user" -r ~/liboqs/build/lib/ "~/liboqs/build/lib/"
echo "Done rsync on liboqs"

echo "Calling rsync"
./runcmd -f "$ipfile" -k "$key" -u "$user" -r ~/go/src/github.com/tcrain/cons/rpcnode "~/go/src/github.com/tcrain/cons/rpcnode"
./runcmd -f "$ipfile" -k "$key" -u "$user" -r ~/go/src/github.com/tcrain/cons/scripts/ "~/go/src/github.com/tcrain/cons/scripts/"
echo "Done rsync"

echo "Updating ulimit"
./runcmd -k "${key}" -u "${user}" -f "${ipfile}" bash ~/go/src/github.com/tcrain/cons/scripts/imagesetup.sh $user

echo "Copying over participant register"
# be sure the process isnt running
ssh -i "${key}" -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" "$user"@"$pregip" "cd ~/go/src/github.com/tcrain/cons/; bash ./scripts/killgo.sh"
# copy
scp -i "${key}" -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" ~/go/src/github.com/tcrain/cons/preg "$user"@"$pregip":~/go/src/github.com/tcrain/cons/preg
echo "Done"