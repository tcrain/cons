# gnuplot -e "filename='tmp'; algs='${AlgNum}'; outfile='${Filename}.png'; ylab=\"Num threads\"; tit=\"${Filename}\"" plot.gp
#filenames="d1.tsv d2.tsv"
#outputfile="test.png"

# mvcons1 SigCreation SigVerification VRFCreation VRFVerification TotalTime
# mvcons1 100         100             100          100             totalTime - (sum others)

set terminal png size width,height enhanced font "Helvetica,12"
set termoption noenhanced
set output outputfile
# set title tit
set key autotitle columnheader
# set key invert reverse Left outside
set xtics rotate by 45 right
set xlabel xlab
set ylabel ylab
# set key left top
set key outside above
set style data histogram
set style histogram rowstacked
set style fill solid border -1
set boxwidth 0.9
set style fill solid 0.3
# set bars front
plot for [file in filenames] file using 2:xtic(1), for [i=3:numCol] '' using i
