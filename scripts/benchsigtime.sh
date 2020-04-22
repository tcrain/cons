
computeMS () {
  in="$1"
  ns=$(echo "$in" | grep "ns/op" | awk '{print $3}')

  bc <<< "scale=4; $ns / 1000000"
}

nArr=(4 8 16 32 48)

for n in "${nArr[@]}"
do

t=$(((n-1) / 3))

for mul in {1..2}
do

thrsh=$(((mul * t) + 1))

echo "-------------------------------------------------"
echo "// n: $n, thrsh: $thrsh"
echo "-------------------------------------------------"

echo Benchmark time to verify and combine Coin2
computeMS "$(go test  ./consensus/auth/sig/ed/... -run=None -bench=BenchmarkEdCoinVerify -thrshn=$n -thrsht=$thrsh)"
echo

echo Benchmark time to combine Coin2
computeMS "$(go test  ./consensus/auth/sig/ed/... -run=None -bench=BenchmarkEdCoinGen -thrshn=$n -thrsht=$thrsh)"
echo


echo Benchmark time to verify and combine Coin1
computeMS "$(go test  ./consensus/auth/sig/bls/... -run=None -bench=BenchmarkBlsCoinVerify -thrshn $n -thrsht $thrsh)"
echo

echo Benchmark time to combine Coin1
computeMS "$(go test  ./consensus/auth/sig/bls/... -run=None -bench=BenchmarkBlsCoinGen -thrshn $n -thrsht $thrsh)"
echo

done
done

echo "Benchmark time to create Coin1 share (BLS thresh sign)"
computeMS "$(go test  ./consensus/auth/sig/bls/... -run=None -bench=BenchmarkBLSCoinSign)"
echo

echo "Benchmark time to verify Coin1 share (BLS thresh verify)"
computeMS "$(go test  ./consensus/auth/sig/bls/... -run=None -bench=BenchmarkBLSShareVerify)"
echo

echo Benchmark time to create Coin2 share
computeMS "$(go test  ./consensus/auth/sig/ed/... -run=None -bench=BenchmarkEdCoinSign)"
echo

echo Benchmark time to verify Coin2 share
computeMS "$(go test  ./consensus/auth/sig/ed/... -run=None -bench=BenchmarkCoinShares)"
echo

echo Benchmark time to ED sig
computeMS "$(go test  ./consensus/auth/sig/ed/... -run=None -bench=BenchmarkSignShares)"
echo

echo Benchmark time to ED verify
computeMS "$(go test  ./consensus/auth/sig/ed/... -run=None -bench=BenchmarkVerifyShares)"
echo

echo "Benchmark time to encrypt message"
computeMS "$(go test  ./consensus/channel/... -run=None -bench=BenchmarkEncrypt)"
echo

echo "Benchmark time to decrypt message"
computeMS "$(go test  ./consensus/channel/... -run=None -bench=BenchmarkDecrypt)"
echo
