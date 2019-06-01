set -e

o="/root/bin/control-$(date +'%Y%m%d-%H%M%S')"
go build -o $o control

# Kill existing control process and run new control binary.
tmux kill-window -t 9
tmux new-window -t 9 "$o"

# User must kill this script within 15 seconds or the script will revert to
# stable version.
sleep 15

# There is likely a problem as user has not terminated the script so revert to
# stable version. 
tmux kill-window -t 9
tmux new-window -t 9 "/root/bin/control"
