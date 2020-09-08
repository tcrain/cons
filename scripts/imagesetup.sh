user=$1

# sudo yum -y update

echo Checking ulimit
if [ "$(ulimit -n)" -eq 10000000 ]
then
    echo "Already set ulimit"
    exit
fi

echo Updating ulimit
echo "
$user soft nofile 10000000
$user hard nofile 10000000
root soft nofile 10000000
root hard nofile 10000000
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
#net.core.netdev_max_backlog=5000
#" | sudo tee -a /etc/sysctl.conf
#
#sudo sysctl -w net.ipv4.route.flush=1
#sudo sysctl -p
#cat /proc/sys/net/ipv4/tcp_*mem