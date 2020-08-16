set -e
set -o pipefail

user=$1
ip=$2
key=$3
goversion=$4
branch=$5
makeuser=$6

if [ "$makeuser" -eq 1 ]
then

    echo "Creating user $user"

    # wait for reboot
    until ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o ConnectTimeout=5 -i $key root@$ip "exit"; do sleep 5; done
    sleep 15

    # Make a user, allow him to sudo without password
    echo Making a user $user;
    ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i $key root@$ip "
    adduser --disabled-password --gecos \"\" $user;
    usermod -aG sudo $user;
    rsync --archive --chown=$user:$user ~/.ssh /home/$user;

    echo \"$user ALL=(ALL) NOPASSWD: ALL
\" | sudo tee /etc/sudoers.d/$user;"

fi

echo "Waiting for node to start"
# wait for reboot
until ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -o ConnectTimeout=5 -i $key $user@$ip "exit"; do sleep 5; done
sleep 15

# Copy ssh key
echo Copying ssh key
# scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i $key $key $user@$ip:~/.ssh/
scp -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i $key $key $user@$ip:~/.ssh/id_rsa

# update packages
# bash ./scripts/imageupdate.sh $user $ip $key

# install packages
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i $key $user@$ip "
echo Downloading packages;
sudo apt-get -y install rsync wget emacs git gnuplot jq pkg-config autoconf automake libtool gcc libssl-dev python3-pytest unzip xsltproc doxygen graphviz make cmake ninja-build python3-pytest-xdist xsltproc;
"

# update packages
# bash ./scripts/imageupdate.sh $user $ip $key

# install rest
ssh -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" -i $key $user@$ip "
echo \"
StrictHostKeyChecking no
\" >> ~/.ssh/generalconfig;
chmod 600 ~/.ssh/generalconfig;

wget -c https://dl.google.com/go/go${goversion}.linux-amd64.tar.gz;
sudo tar -C /usr/local -xzf go${goversion}.linux-amd64.tar.gz;

echo Updating env in .profile;
echo '
export GOROOT=/usr/local/go
export GOPATH=~/go
export PATH=\$PATH:\$GOROOT/bin
export PATH=\$PATH:\$GOPATH/bin
export LD_LIBRARY_PATH=\$LD_LIBRARY_PATH:/usr/local/lib
export PKG_CONFIG_PATH=\$PKG_CONFIG_PATH:\$GOPATH/src/github.com/open-quantum-safe/liboqs-go/.config
' >> ~/.profile;

source ~/.profile;

echo Installing libqos;
git clone -b master https://github.com/open-quantum-safe/liboqs.git;
cd liboqs;
mkdir build && cd build;
cmake -GNinja -DBUILD_SHARED_LIBS=ON ..;
ninja;
sudo ninja install;

echo Installing dlv;
go get github.com/go-delve/delve/cmd/dlv;

echo Downloading project from git;
ssh-keyscan github.com >> ~/.ssh/known_hosts;

mkdir -p ~/go/src/github.com/open-quantum-safe/;
cd ~/go/src/github.com/open-quantum-safe/;
git clone https://github.com/open-quantum-safe/liboqs-go;

mkdir -p ~/go/src/github.com/tcrain/;
cd ~/go/src/github.com/tcrain;
git clone git@github.com:tcrain/cons.git;

echo Checking out branch $branch;
cd cons;
git checkout $branch;

echo Running go get;
go get -t -u -v github.com/tcrain/cons/...;

echo Updating file limit;
bash ./scripts/imagesetup.sh $user;

echo Building;
bash ./scripts/buildgo.sh 1;
"

# scp -i $key ./scripts/imagesetup.sh $user@$ip:~/imagesetup.sh
