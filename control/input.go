package main

/*

#include <linux/hidraw.h>
#include <sys/ioctl.h>

int _get_raw_phys(int fd, void * data) {
	return ioctl(fd, HIDIOCGRAWNAME(256), data);
}

*/
import "C"

import (
	"encoding/binary"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func handleDev(file string, eventChan chan event) error {
	if !strings.HasPrefix(file, "hidraw") {
		return nil
	}

	// Open raw HID and get name.
	f, err := os.Open("/dev/" + file)
	if err != nil {
		return err
	}
	data := make([]byte, 256)
	l := C._get_raw_phys(C.int(f.Fd()), unsafe.Pointer(&data[0]))
	name := string(data[:l-1])

	// Handle keyboard. Return error if device does not have ErgoBlue prefix.
	prefix := "ErgoBlue "
	if !strings.HasPrefix(name, prefix) {
		return errors.New("unrecognized raw HID input")
	} else {
		go handleKeyboard(f, strings.TrimPrefix(name, prefix), eventChan)
		return nil
	}
}

func handleInput() error {
	// Initialize channel for keyboard events and handle concurrently.
	eventChan := make(chan event)
	s := &state{device: defaultDevice}
	go s.handleEvent(eventChan)

	// Use inotify to watch /dev for new inputs. Allocate a sufficient buffer
	// for inotify event data.
	fd, err := syscall.InotifyInit()
	if err != nil {
		return err
	}
	if _, err := syscall.InotifyAddWatch(fd, "/dev", syscall.IN_CREATE); err != nil {
		return err
	}

	// Handle all existing files in /dev. This is done after rather than before
	// initializing inotify so we see the HID file even if the Bluetooth
	// connection is established while this is running.
	files, err := ioutil.ReadDir("/dev")
	if err != nil {
		return err
	}
	for _, value := range files {
		if err := handleDev(value.Name(), eventChan); err != nil {
			return err
		}
	}

	data := make([]byte, 256*syscall.SizeofInotifyEvent)
	for {
		n, err := syscall.Read(fd, data)
		if err != nil {
			return err
		}

		for i := 0; i < n; {
			// Determine length of filename and advance the read pointer. The
			// name is padded with 0x00 bytes that need to be removed.
			end := i + syscall.SizeofInotifyEvent
			size := int(binary.LittleEndian.Uint32(data[end-4 : end]))
			i = end + size
			file := strings.TrimRight(string(data[end:i]), "\x00")
			if err := handleDev(file, eventChan); err != nil {
				return err
			}
		}
	}
}
