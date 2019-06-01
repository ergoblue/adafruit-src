# Vendor keycode.h from qmk/qmk_firmware, which contains HID keycodes.
wget -q https://raw.githubusercontent.com/qmk/qmk_firmware/8df044b86828a374fc2c872c2bedc2f4b567f5bf/tmk_core/common/keycode.h

# Vendor usbkbd.c from torvalds/linux, which contains Linux keyboard event
# codes. Store the useful portion in usbkbd.h.
wget -q https://raw.githubusercontent.com/torvalds/linux/cc10ad25bbca3d2925adc32d51cb7a10b837d32c/drivers/hid/usbhid/usbkbd.c
sed -n "/usb_kbd_keycode/,//p" usbkbd.c | sed '/}/q' > usbkbd.h
rm usbkbd.c
sed -i "s/^static const //" usbkbd.h
