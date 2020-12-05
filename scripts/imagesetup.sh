user=$1

# sudo yum -y update

echo Checking ulimit
if [ "$(ulimit -n)" -eq 1000000 ]
then
    echo "Already set ulimit"
    exit
fi

echo Updating ulimit
echo "
$user soft nofile 1000000
$user hard nofile 1000000
root soft nofile 1000000
root hard nofile 1000000
" | sudo tee -a /etc/security/limits.conf

# logout and back in
# check the open file limit was increased
ulimit -n

#echo Increasing network buffer sizes
#echo "net.core.rmem_max=8388608
#net.core.wmem_max=8388608
#net.core.rmem_default=65536
#net.core.wmem_default=65536
#net.ipv4.tcp_rmem=4096 87380 8388608
#net.ipv4.tcp_wmem=4096 65536 8388608
#net.ipv4.tcp_mem=8388608 8388608 8388608
#" | sudo tee -a /etc/sysctl.conf
#

echo Increase TCP backlog
echo "net.ipv4.tcp_max_syn_backlog=4096
net.core.netdev_max_backlog=16384
net.core.somaxconn = 4096" | sudo tee -a /etc/sysctl.conf

sudo sysctl -w net.ipv4.route.flush=1
sudo sysctl -p
cat /proc/sys/net/ipv4/tcp_max_syn_backlog
#cat /proc/sys/net/ipv4/tcp_*mem