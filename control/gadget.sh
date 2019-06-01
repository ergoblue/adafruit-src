set -e

# Load necessary kernel modules.
modprobe libcomposite dwc2

cd /sys/kernel/config/usb_gadget
if [ ! -d ergoblue ]; then
	# Create gadget.
	mkdir ergoblue && cd ergoblue

	# Set vendor and product. This creates a Multifunction Composite Gadget
	# under the Linux Foundation per https://goo.gl/3hqzzF.
	echo 0x1d6b > idVendor
	echo 0x0104 > idProduct

	# Define support for USB 2.
	echo 0x0200 > bcdUSB # USB2

	# Set device manufacturer and product. Per https://goo.gl/fGfYEj, 0x409
	# represents United States English.
	mkdir -p strings/0x409
	echo "Xudong Zheng" > strings/0x409/manufacturer
	echo "ErgoBlue" > strings/0x409/product

	# Per "Device Class Definition for Human Interface Devices", set protocol to
	# 1 for keyboard. Set subclass to 0 to designate that the device does not
	# support the boot HID protocol.
	mkdir -p functions/hid.usb0
	echo 1 > functions/hid.usb0/protocol
	echo 0 > functions/hid.usb0/subclass

	# Set HID report.
	echo 8 > functions/hid.usb0/report_length
	echo "$1" | base64 -d > functions/hid.usb0/report_desc

	mkdir -p configs/c.1
	ln -s functions/hid.usb0 configs/c.1/
	ls /sys/class/udc > UDC
fi
