set -e
set -o pipefail

vars=()

while read -r line
do
    vars+=("$line")
done < .lastrun
pregip=${vars[0]}
oldtofolder=${vars[1]}
benchid=${vars[2]}
nodecounts=${vars[3]}
user=${vars[4]}
ipfile=${vars[5]}
key=${vars[6]}
pregport=${vars[7]}
nwtest=${vars[8]}
project=${vars[9]}
credentialPath=${vars[10]}
singleZoneCmd=${vars[11]}
regions=${vars[12]}
tofolders=${vars[13]}

alloutputfolders=""

# Get ips
bash ./scripts/cloudscripts/getips.sh "$singleZoneCmd" "$regions" "$project" "$credentialPath"

foundTo=false

# Check which test options folders have already been run
for tofolder in $tofolders
do

  if [ "$foundTo" == true ]
  then
    benchid=$(date +"%m-%d-%y_%T")
  fi

  if [ "$tofolder" == "$oldtofolder" ]
  then
    foundTo=true
  fi

  if [ "$foundTo" == false ]
  then
    echo Already ran test configs folder "$tofolder", skipping
    continue
  fi

echo "$pregip
$tofolder
$benchid
$nodecounts
$user
$ipfile
$key
$pregport
$nwtest
$project
$credentialPath
$singleZoneCmd
$regions
$tofolders" > .lastrun

alloutputfolders="$alloutputfolders ./benchresults/$benchid"

echo Running Bench "$tofolder"
if ! bash ./scripts/Bench.sh "$pregip" "$tofolder" "$benchid" "$nodecounts" "$user" "$ipfile" "$key" "$pregport" "$nwtest"
then

  for i in {1..5}
  do
    echo Benchmark failed, going to retry after 30 seconds
    sleep 30s

    echo Retrying Bench
    if bash ./scripts/Bench.sh "$pregip" "$tofolder" "$benchid" "$nodecounts" "$user" "$ipfile" "$key" "$pregport" "$nwtest"
    then
      break
    fi

    if [ $i -eq 5 ]
    then
      echo "Failed benchmark too many times, exiting, outpufolders are $alloutputfolders"
      exit 1
    fi

  done

fi

done

echo "Done running benches, output folders are $alloutputfolders"