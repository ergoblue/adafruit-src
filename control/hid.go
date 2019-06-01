package main

import (
	"encoding/binary"
	"io"
)

// The main keyboard descriptor was developed from https://goo.gl/xjNxy3. The
// consumer page descriptor was developed from https://goo.gl/qEeXj7. See
// https://goo.gl/RYBXdb for a comprehensive guide on HID descriptors.
// https://goo.gl/HZydaN provides a tool to visualize the descriptor.
var keyboardReport = []byte{
	0x05, 0x01, // Usage Page (Generic Desktop Ctrls)
	0x09, 0x06, // Usage (Keyboard)
	0xa1, 0x01, // Collection (Application)
	0x85, 0x01, // Report ID (1)
	0x95, 0x08, // Report Count (8)
	0x75, 0x01, // Report Size (1)
	0x05, 0x07, // Usage Page (Kbrd/Keypad)
	0x19, 0xe0, // Usage Minimum (0xE0)
	0x29, 0xe7, // Usage Maximum (0xE7)
	0x15, 0x00, // Logical Minimum (0)
	0x25, 0x01, // Logical Maximum (1)
	0x81, 0x02, // Input
	0x95, 0x06, // Report Count (6)
	0x75, 0x08, // Report Size (8)
	0x15, 0x00, // Logical Minimum (0)
	0x25, 0xff, // Logical Maximum (255)
	0x05, 0x07, // Usage Page (Kbrd/Keypad)
	0x19, 0x00, // Usage Minimum (0x00)
	0x29, 0xff, // Usage Maximum (0xFF)
	0x81, 0x00, // Input
	0xc0, // End Collection
	0x05, 0x0c, // Usage Page (Consumer)
	0x09, 0x01, // Usage (Consumer Control)
	0xa1, 0x01, // Collection (Application)
	0x85, 0x02, // Report ID (2)
	0x95, 0x01, // Report Count (1)
	0x75, 0x10, // Report Size (16)
	0x15, 0x01, // Logical Minimum (1)
	0x26, 0x9c, 0x02, // Logical Maximum (668)
	0x19, 0x01, // Usage Minimum (Consumer Control)
	0x2a, 0x9c, 0x02, // Usage Maximum (AC Distribute Vertically)
	0x81, 0x00, // Input
	0xc0, // End Collection
}

type hidWriter struct {
	io.Writer
	keys      map[uint8]bool
	modifiers uint8
}

func (w *hidWriter) desktopWrite() {
	data := make([]byte, 0, 8)
	data = append(data, 0x01)
	data = append(data, w.modifiers)
	for key := range w.keys {
		data = append(data, key)
	}
	for i := len(w.keys); i < 6; i++ {
		data = append(data, 0)
	}
	w.Write(data)
}

func (w *hidWriter) pressDesktop(key uint8) {
	if isModifier(key) {
		w.modifiers |= 1 << getModifierIndex(key)
	} else if len(w.keys) < 6 {
		w.keys[key] = true
	} else {
		return
	}
	w.desktopWrite()
}

func (w *hidWriter) releaseDesktop(key uint8) {
	if isModifier(key) {
		w.modifiers &= ^(1 << getModifierIndex(key))
	} else if _, ok := w.keys[key]; ok {
		delete(w.keys, key)
	} else {
		return
	}
	w.desktopWrite()
}

func (w *hidWriter) sendConsumer(key uint16) {
	data := make([]byte, 3)
	data[0] = 0x02
	binary.LittleEndian.PutUint16(data[1:], key)
	w.Write(data)
}

func newHIDWriter(w io.Writer) *hidWriter {
	return &hidWriter{Writer: w, keys: make(map[uint8]bool)}
}
