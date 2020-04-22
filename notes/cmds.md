tail -f testcoinbench.out ---disable-inotify | grep --line-buffered panic

go run ./cmd/gento/gento.go

./scripts/Bench.sh 127.0.0.1 ./testconfigs/s-coinbyzpresets/ "" "4" none ipfile none 4534 0

find ./scripts/ -type f -exec sed -i 's/\r$//' {} +