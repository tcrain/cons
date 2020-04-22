# gnuplot -e "filename='tmp'; algs='${AlgNum}'; outfile='${Filename}.png'; ylab=\"Num threads\"; tit=\"${Filename}\"" plot.gp
#filenames="d1.tsv d2.tsv"
#outputfile="test.png"

set terminal png size width,height enhanced font "Helvetica,12"
set output outputfile
set termoption noenhanced
# set title tit
set xlabel xlab
set ylabel ylab
set xtics rotate by 45 right
# set key left top
set key outside above
set style data histogram
set style histogram errorbars
set style fill solid border -1
set boxwidth 0.9
set style fill solid 0.3
set bars front
plot for [file in filenames] file using 3:2:4:xtic(1) title word(system('head -1 '.file), 2)
