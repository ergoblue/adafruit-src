//go:generate bash vendor.sh
//go:generate bash setup.sh

package main

import "log"

// Define MAC addresses of the two keyboard halves. Note that these are in the
// reverse order of the printable address.
const (
	leftMAC  = "000000000000"
	rightMAC = "111111111111"
)

const defaultDevice = "abc123"

// Initialize Bluetooth devices. Gadget and uinput are initialized in main().
// Define the temporary Bluetooth device with an empty MAC address.
var devices = map[string]config{
	"def123": {writer: newBlueZWriter("00:00:00:00:00:00"), platform: platMacOS, qwerty: true},
	"ghi123": {writer: newBlueZWriter("00:00:00:00:00:00"), platform: platMacOS},
	"jkl123": {writer: newBlueZWriter("00:00:00:00:00:00"), platform: platAndroid},
	"mno123": {writer: newBlueZWriter("00:00:00:00:00:00"), platform: platWindows, qwerty: true},
	"tmp123": {writer: newBlueZWriter(""), platform: platMacOS},
}

func main() {
	if w, err := newGadgetWriter(); err != nil {
		log.Fatal(err)
	} else {
		devices["abc123"] = config{writer: w, platform: platLinux}
	}

	// Define writer for outputting to the controller itself. This device is
	// called uin123 since it goes through /dev/uinput and the physical device
	// may vary.
	if w, err := newUinputWriter(); err != nil {
		log.Fatal(err)
	} else {
		devices["uin123"] = config{writer: w, platform: platLinux}
	}

	// Setup Bluetooth listener. This will return once everything is setup.
	if err := handleBlueZ(); err != nil {
		log.Fatal(err)
	}

	// Handle inputs. This will block until there is a major error.
	if err := handleInput(); err != nil {
		log.Fatal(err)
	}
}
