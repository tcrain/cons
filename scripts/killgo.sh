if [ -z "$GOPATH" ]
then
  GOPATH=~/go/
fi

pkill preg
pkill rpcnode
bash "$GOPATH"/src/github.com/tcrain/cons/scripts/cleanstuff.sh
