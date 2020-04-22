benchfolder=$1

# This gets all in the generalconfig file names in the benchfolder
for f in "${benchfolder}"/*.json
do
    nxt=$(echo $f | sed "s#^.*${benchfolder}/\+\(.*\)\.json#\1#")
    filename="$filename$nxt "
done
echo "$filename"
