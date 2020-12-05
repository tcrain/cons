# gnuplot -e "filename='tmp'; algs='${AlgNum}'; outfile='${Filename}.png'; ylab=\"Num threads\"; tit=\"${Filename}\"" plot.gp
#filenames="d1.tsv d2.tsv"
#outputfile="test.png"

set terminal png size width,height enhanced font "Helvetica,12"
set output outputfile
set termoption noenhanced
# set title tit
# set xtics rotate by 45 right
# set key left top
# set key outside above
set style data histogram
set style histogram errorbars
set style fill solid border -1
set boxwidth 0.9
set style fill solid 0.3
set bars front

set multiplot layout 2,2 margins .15, .85, .2, .95
set key at screen 0.5, 0.05 center vertical height 1 nobox maxrows 2

set ylabel "Time Per Decision (seconds)"
plot for [file in filenames1] file using ($3/1000):($2/1000):($4/1000):xtic(1) title word(system('head -1 '.file), 2)

# unset ylabel
unset ytics
set y2tics mirror
set ylabel "Time Per Consensus (seconds)"
plot for [file in filenames2] file using ($3/1000):($2/1000):($4/1000):xtic(1) title word(system('head -1 '.file), 2) axes x1y2

# unset y2label
unset y2tics
set ytics
set xlabel xlab3
set ylabel "Data Sent (kilobytes)"
plot for [file in filenames3] file using ($3/1000):($2/1000):($4/1000):xtic(1) title word(system('head -1 '.file), 2)

# unset ylabel
unset ytics
set y2tics mirror
set xlabel xlab4
set ylabel "Messages Signed"
plot for [file in filenames4] file using ($3):($2):($4):xtic(1) title word(system('head -1 '.file), 2) axes x1y2
