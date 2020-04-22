
# run from home directory so doesnt install module
go get github.com/go-delve/delve/cmd/dlv

# see flags in buildgo.sh
go build -gcflags="all=-N -l" <file>

dlv attach pid exec

break cons_state.go:263

p cs.memberCheckerState.LocalIndex

p cs.memberCheckerState.consItemsMap[10].MsgState.auxValues
