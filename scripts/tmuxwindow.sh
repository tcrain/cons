# takes as input a name for the tmux session, plus a list of commands, each opened in a different tmux window

name=$1

tmux kill-session -t "$name"
tmux new -s "$name" -d

shift
while (( "$#" ))
do
  tmux send-keys -t "$name" "$1" C-m
  tmux new-window -t "$name"

  shift
done

tmux kill-window -t "$name"
tmux attach -t "$name"