package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"log"
	"net"
	"os/exec"
	"text/template"
	"time"

	"golang.org/x/sys/unix"
)

// Create map for determining whether a MAC address is authorized and getting
// its corresponding channel.
var blChan = make(map[string]chan []byte)

// Define channel for user to reset all temporary Bluetooth connections.
var tmpChan = make(chan bool)

type blueZWriter chan []byte

func (w blueZWriter) Write(p []byte) (int, error) {
	// Construct Bluetooth message. There must be a 0xa1 byte preceeding the
	// HID data.
	data := make([]byte, 0, 9)
	data = append(data, 0xa1)
	data = append(data, p...)

	// Write to channel if possible. Allow 5ms for the message to be read.
	// Typically only 1.5ms is necessary. If unsuccessful, ignore message as
	// the device must not be connected.
	select {
	case w <- data:
	case <-time.After(5 * time.Millisecond):
	}

	return len(p), nil
}

func newBlueZWriter(mac string) *hidWriter {
	c := make(chan []byte)
	blChan[mac] = c
	return newHIDWriter(blueZWriter(c))
}

func formatMAC(addr [6]uint8) string {
	var data [6]byte
	for key, value := range addr {
		data[5-key] = value
	}
	return net.HardwareAddr(data[:]).String()
}

func serveControl(nfd int, c chan []byte) {
	// While we do not make use of the control endpoint, the client expects us
	// to keep this open. This function returns when unix.Read() returns,
	// presumably after the client has closed its connection to both ports.
	data := make([]byte, 1)
	unix.Read(nfd, data)
}

func serveInterrupt(nfd int, c chan []byte) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Send HID data to device. When the context is cancelled, return since the
	// device must have closed the connection.
	go func() {
		for {
			select {
			case data := <-c:
				unix.Write(nfd, data)
			case <-ctx.Done():
				return
			}
		}
	}()

	// When the device disconnects, unix.Read() will return. This is because we
	// do not expect the device to send any data. Cancel the context to
	// terminate the goroutine above.
	data := make([]byte, 1)
	unix.Read(nfd, data)
}

func listenL2CAP(psm uint16, h func(int, chan []byte)) error {
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_SEQPACKET, unix.BTPROTO_L2CAP)
	if err != nil {
		return err
	}
	if err := unix.Bind(fd, &unix.SockaddrL2{PSM: psm}); err != nil {
		return err
	}

	// Support up to 16 concurrent connections.
	if err := unix.Listen(fd, 16); err != nil {
		return err
	}

	// Create lock to limit each PSM to one temporary connection.
	tmpLock := make(chan bool, 1)

	go func() {
		for {
			nfd, sa, err := unix.Accept(fd)
			if err != nil {
				continue
			}
			addr := formatMAC(sa.(*unix.SockaddrL2).Addr)
			go func() {
				defer unix.Close(nfd)

				// Handle connection directly if MAC address is recognized.
				if c, ok := blChan[addr]; ok {
					h(nfd, c)
					return
				}

				// Return if there is already a temporary connection.
				select {
				case tmpLock <- true:
					defer func() {
						<-tmpLock
					}()
				default:
					return
				}

				// Handle with the channel for temporary connections, which uses
				// an empty string as the MAC address for the blChan map key.
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					h(nfd, blChan[""])
					cancel()
				}()

				// If the user decides to reset all temporary connections,
				// tmpChan will become writable. The function should return once
				// the handler returns or temporary connections are reset.
				select {
				case tmpChan <- true:
					return
				case <-ctx.Done():
					return
				}

			}()
		}
	}()

	return nil
}

func handleBlueZ() error {
	// Make device discoverable.
	if err := exec.Command("hciconfig", "hci0", "piscan").Run(); err != nil {
		return err
	}

	// Use Python for Bluetooth service discovery.
	tpl, err := template.ParseFiles("src/control/sdp.xml")
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	tpl.Execute(buf, hex.EncodeToString(keyboardReport))
	cmd := exec.Command("python3", "src/control/sdp.py")
	cmd.Stdin = buf
	go func() {
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		} else {
			log.Fatal("sdp.py exited")
		}
	}()

	// Handle control on PSM 17 and interrupt on PSM 19. These values are
	// defined on https://goo.gl/sHJyeB.
	if err := listenL2CAP(17, serveControl); err != nil {
		return err
	}
	if err := listenL2CAP(19, serveInterrupt); err != nil {
		return err
	}

	return nil
}
