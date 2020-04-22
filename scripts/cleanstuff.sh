if [ -z "$GOPATH" ]
then
  GOPATH=~/go/
fi

echo 'Running find ~/go/src/github.com/tcrain/cons/ -type f -name '*_test*.store' -exec rm {} +'
find "$GOPATH"/src/github.com/tcrain/cons/ -type f -name '*_test*.store' -exec rm {} +

echo 'Running find ~/go/src/github.com/tcrain/cons/ -type f -name 'memprof*[0-9]*_proc[0-9]*.out' -exec rm {} +'
find "$GOPATH"src/github.com/tcrain/cons/ -type f -name 'memprof*[0-9]*_proc[0-9]*.out' -exec rm {} +

echo 'Running find ~/go/src/github.com/tcrain/cons/ -type f -name 'cpuprof*[0-9]*_proc[0-9]*.out' -exec rm {} +'
find "$GOPATH"/src/github.com/tcrain/cons/ -type f -name 'cpuprof*[0-9]*_proc[0-9]*.out' -exec rm {} +
