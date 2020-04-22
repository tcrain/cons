set -e
set -o pipefail
option=$1
newValue=$2
tofolder=$3

echo "Setting option: $option to $newValue in testconfigs"

for f in ./"${tofolder}"/*.json
do
    # for when the line ends with a comma
    sed -i "s/\"$option\".*,$/\"$option\": $newValue,/g" $f
    # for when the line doesn't end with a comma
    sed -i "s/\"$option\".*[^,]$/\"$option\": $newValue/g" $f
done
