set -e
set -o pipefail

benchid=$1
tofilenames=$2
ipfile=$3
pregip=$4
tofolder=$5
nc=$6

benchfolder="./benchresults/$benchid"

for f in $tofilenames
do

  logname="./benchresults/$benchid/cons_${f}.log"

  f="${tofolder}${f}.json"

  constypes=$(./gettestconsids -o "${f}")

  echo "Got cons types $constypes for test file $f"

  for ctype in $constypes
  do

    bash ./scripts/benchscripts/updatetestconfigs.sh ConsType "$ctype" "$tofolder"

    # Check if the config is valid
    echo "Checking test config $f"
    if ! ./localrun -o "$f" -c
    then
        echo "Test config invalid, skipping"
        continue
    fi

    touch ./benchresults/"$benchid"/finished
    if grep -q "${f}_${ctype}_$nc" ./benchresults/"$benchid"/finished
    then
      echo "Already ran test, continuing"
      continue
    fi

    # we add the test generalconfig to the results folder
    # cp $f ./benchresults/$benchid/
    # results filename is results_{testID}_{nodes}_{constype}
    #filename=$(echo $f | sed 's#\.\/testconfigs\/\(.*\)\.json#\1#')

    echo
    echo "Running: ./rpcbench -o $f -f $ipfile -p $pregip -r $benchfolder" | tee -a "$logname"
    # cat "$f" | tee -a "$logname"
    tee -a "$logname" < "$f"
    sleep 2
    ./rpcbench -o "$f" -f "$ipfile" -p "$pregip" -r "$benchfolder" 2>&1 | tee -a "$logname"

    # Record that we finished this benchmark
    echo "${f}_${ctype}_$nc" >> ./benchresults/"$benchid"/finished

  done
done