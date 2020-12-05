#!/bin/bash

name=$1
cmd1=$2
cmd2=$3
cmd3=$4
cmd4=$5

tmux kill-window -t "$name"
tmux new -s "$name" -d
tmux send-keys -t "$name" "$cmd1" C-m

tmux split-window -v -t "$name"
tmux send-keys -t "$name" "$cmd2" C-m

tmux split-window -h -t "$name"
tmux send-keys -t "$name" "$cmd3" C-m

tmux select-pane -U
tmux split-window -h -t "$name"
tmux send-keys -t "$name" "$cmd4" C-m

tmux attach -t "$name"
