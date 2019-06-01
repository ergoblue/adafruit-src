set -e

# Load $PATH variable using /etc/profile.
source /etc/profile

# Use bash as shell.
SHELL=/bin/bash

# Wait for the Bluetooth device to be up before further initialization.
while true; do
	if [ "$(hciconfig)" == "" ]; then
		sleep 1
	else
		break
	fi
done

# Stop BlueZ if it's running.
service bluetooth stop

# Initialize Bluetooth device.
hciconfig hci0 up

# Start tmux session.
tmux new-session -d

# Start BlueZ. We need the HID over GATT (hog) and GAP plugins for keyboard
# input. We need the time plugin to function as a peripheral for output. Run in
# tmux for easy debugging.
tmux new-window -t 8 "bluetoothd --nodetach --debug -p hog,gap,time"

# By default, run /root/bin/control on startup. That version is expected to be
# reliable. We will often run a newer version manually and replace the stable
# version over time.
tmux new-window -t 9 "/root/bin/control"
