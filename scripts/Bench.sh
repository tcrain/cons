#!/bin/bash
set -e
set -o pipefail

if [ "$#" -lt 2 ]
then
    echo "Usage: \"./scripts/Bench.sh participant-register-ip testoptions-folder benchid \"node-counts\" \"cons-types\" user ipfile ssh-key  participant-register-port {Network test (0 for no, 1 for yes, default 1)} \""
    exit
fi

pregip=${1}
tofolder=${2}
benchid=$(date +"%m-%d-%y_%T")"-"$(basename $tofolder)
benchid=${3:-$benchid}
nodecounts=${4:-4}
user=${5:-$BENCHUSER}
ipfile=${6:-./benchIPfile}
key=${7:-$KEYPATH}
pregport=${8:-4534}
nwtest=${9:-1}
enableprofile=${10:-$PROF}

if [ "$enableprofile" == "" ]
then
  enableprofile=0
fi

if [ "$nwtest" -ne 1 ]
then
    pregip="127.0.0.1"
    pregport="4534"
fi

# proposalsizes="10"
#nodecounts="4 8 12"
#constypes="0 1"

echo
echo "Running consensus benchmarks"
echo

benchfolder="./benchresults/$benchid"

tofilenames=$(bash ./scripts/graphscripts/gettestoptionsnames.sh "./$tofolder") # use all the generalconfig files in the folder
#tofilenames="bls_multi_p2p ec_all2all"
#tofilenames="ec_all2all  ecproofs_all2all ed_rnd"
#tofilenames="ed_rnd"

echo Running setup
echo bash ./scripts/benchscripts/benchsetup.sh "$user" "$ipfile" "$key" $pregip "$pregport" "$benchfolder" "$nwtest" "$tofolder" "$enableprofile"
bash ./scripts/benchscripts/benchsetup.sh "$user" "$ipfile" "$key" $pregip "$pregport" "$benchfolder" "$nwtest" "$tofolder" "$enableprofile"

echo Running bench
bash ./scripts/benchscripts/runnodeconfig.sh "$nodecounts" "$benchid" "$tofilenames" "$ipfile" "${pregip}:${pregport}" "$tofolder" | tee -a "${benchfolder}"/full.log

echo Finished successfully

echo Copying gen sets file
mkdir -p "$benchfolder"/gensets/
cp "$tofolder"/gensets/gensets.json "$benchfolder"/gensets/
echo "Results are in $benchfolder"

# make the graphs
echo Generating results
echo go run ./cmd/genresults/genresults.go -o "$benchfolder"
go run ./cmd/genresults/genresults.go -o "$benchfolder"

echo Done generating results
bash ./scripts/killgo.sh
