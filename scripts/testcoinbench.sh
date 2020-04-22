set -e
set -o pipefail

tofolder="testconfigs/"

echo New Test > testcoinbench.out

for d in "${tofolder}"*/
do
  echo Running folder $d
  ./scripts/Bench.sh 127.0.0.1 "$d" "" "4" none ipfile none 4534 0 2>&1 | tee -a testcoinbench.out
  echo Done running folder $d
done
