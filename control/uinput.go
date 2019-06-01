package main

/*

#include <linux/uinput.h>
#include <string.h>
#include <unistd.h>

#include "usbkbd.h"

void _write_event(int fd, int type, int code, int val) {
	struct input_event ev = {.type = type, .code = code, .value = val};

	// Since write() has warn_unused_result, we must store its return value as
	// some variable to suppress the compiler warning.
	int n = write(fd, &ev, sizeof(ev));
}

void _set_keybit(int fd, unsigned char ev) {
	ioctl(fd, UI_SET_KEYBIT, ev);
}

// Initialize uinput device. For simplicity, use the same vendor and product ID
// as the OTG gadget.
void _init_uinput(int fd) {
	struct uinput_setup usetup;
	ioctl(fd, UI_SET_EVBIT, EV_KEY);
	memset(&usetup, 0, sizeof(usetup));
	usetup.id.bustype = BUS_USB;
	usetup.id.vendor = 0x1d6b;
	usetup.id.product = 0x0104;
	strcpy(usetup.name, "ErgoBlue");
	ioctl(fd, UI_DEV_SETUP, &usetup);
	ioctl(fd, UI_DEV_CREATE);
}

*/
import "C"

import "os"

// Use uinput to use the keyboard with the controller itself. This code is based
// on https://goo.gl/XyNyhr.
type uinputWriter struct {
	fd C.int
}

func (w uinputWriter) write(key uint8, value C.int) {
	if evkey := C.usb_kbd_keycode[key]; evkey != 0 {
		C._write_event(w.fd, C.EV_KEY, C.int(evkey), value)
		C._write_event(w.fd, C.EV_SYN, C.SYN_REPORT, 0)
	}
}

func (w uinputWriter) pressDesktop(key uint8) {
	w.write(key, 1)
}

func (w uinputWriter) releaseDesktop(key uint8) {
	w.write(key, 0)
}

func (w uinputWriter) sendConsumer(key uint16) {}

func newUinputWriter() (uinputWriter, error) {
	var w uinputWriter

	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0666)
	if err != nil {
		return w, err
	}
	w.fd = C.int(f.Fd())

	for _, value := range C.usb_kbd_keycode {
		if value != 0 {
			C._set_keybit(w.fd, value)
		}
	}
	C._init_uinput(w.fd)

	return w, nil
}
