package main

import (
	"encoding/base64"
	"os"
	"os/exec"
	"time"
)

type gadgetWriter struct {
	f *os.File
}

func (w gadgetWriter) Write(p []byte) (int, error) {
	// Allow 5ms for write to complete. Most should occur well under 1ms. This
	// is necessary because if there is no device reading from the gadget, the
	// write operation would otherwise block and make it impossible to use with
	// other devices.
	w.f.SetWriteDeadline(time.Now().Add(5*time.Millisecond))
	return w.f.Write(p)
}

func newGadgetWriter() (*hidWriter, error) {
	report64 := base64.StdEncoding.EncodeToString(keyboardReport)
	cmd := exec.Command("bash", "src/control/gadget.sh", report64)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	f, err := os.OpenFile("/dev/hidg0", os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	return newHIDWriter(gadgetWriter{f}), nil
}
