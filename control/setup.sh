# Set device name and class. The device class can be calculated using
# https://goo.gl/574f1T. The device is currently only a keyboard. Typical HID
# devices run with limited discoverable mode but this is always discoverable.
name="ErgoBlue $(hostname | tr a-z A-Z)"
sed -i "s/#Name = .*/Name = $name/" /etc/bluetooth/main.conf
sed -i "s/#Class = .*/Class = 0x000540/" /etc/bluetooth/main.conf

# Allow Raspberry Pi to be used as USB HID device.
echo "dtoverlay=dwc2" >> /boot/config.txt

# Run control/boot.sh when server starts.
echo "@reboot bash /root/src/control/boot.sh" | crontab -

# Install Python D-Bus package for interfacing with BlueZ.
apt-get install -y python3-dbus

# Reboot for changes to take effect.
reboot
