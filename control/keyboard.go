package main

// #include "keycode.h"
import "C"

import (
	"encoding/binary"
	"io"
	"os"
	"time"
)

type lock int

type platform int

const (
	lockNone lock = iota
	lockMod
	lockPowerOff
	lockReboot

	platAndroid platform = iota
	platLinux
	platMacOS
	platWindows
)

type writer interface {
	pressDesktop(uint8)
	releaseDesktop(uint8)
	sendConsumer(uint16)
}

type config struct {
	writer   writer
	qwerty   bool
	platform platform
}

type event struct {
	data []byte
	left bool
}

type state struct {
	device string

	// Track whether each key is currently actuated.
	keys [76]bool

	// Track layer. If the layer is a one-shot layer, the user will return to
	// the default layer after the next key that does not change the layer.
	layer    int
	layerOSL bool

	// Track whether a locking key is active.
	lock lock

	// Track modifier state and whether each modifier is locked. Locked
	// modifiers are not released until it is pressed again. Even though we only
	// need 5 out of the 8 modifiers, allow all 8 to simplify code.
	modifiers, modLocks [8]bool
}

func isModifier(key uint8) bool {
	return key >= C.KC_LCTRL && key <= C.KC_RGUI
}

func getModifierIndex(key uint8) uint8 {
	return key - C.KC_LCTRL
}

func (s *state) releaseModifier(i int) {
	devices[s.device].writer.releaseDesktop(C.KC_LCTRL + uint8(i))
	s.modifiers[i], s.modLocks[i] = false, false
}

func (s *state) handleKey(key uint8) {
	// For modifiers, toggle the state. For regular keys, press and release and
	// release unlocked modifiers. If key is 0, there is no need to send any
	// data but unlocked modifiers should be unset.
	if isModifier(key) {
		i := getModifierIndex(key)
		if s.modifiers[i] {
			s.releaseModifier(int(i))
		} else {
			devices[s.device].writer.pressDesktop(key)
			s.modifiers[i], s.modLocks[i] = true, s.lock == lockMod
		}
	} else {
		if key != 0 {
			devices[s.device].writer.pressDesktop(key)

			// Per https://bit.ly/2Uoy9yG, there must be a minor delay between
			// pressing and releasing the Caps Lock key on MacOS.
			if key == C.KC_CAPSLOCK && devices[s.device].platform== platMacOS {
				time.Sleep(100 * time.Millisecond)
			}

			devices[s.device].writer.releaseDesktop(key)
		}
		for key, value := range s.modifiers {
			if value && !s.modLocks[key] {
				s.releaseModifier(key)
			}
		}
	}
}

func (s *state) handleEvent(eventChan chan event) {
	for {
		ev := <-eventChan

		// Shift by 8 bits to get rid of the leading 0x01 in the original data.
		value := binary.LittleEndian.Uint64(ev.data) >> 8

		// On the left hand side, handle the columns in reverse order since the
		// keymap is defined from top to bottom and left to right. The right
		// hand side keys should start with an offset of 38.
		var i, k0, k1, dk int
		if ev.left {
			k0, k1, dk = 6, -1, -1
		} else {
			i, k0, k1, dk = 38, 0, 7, 1
		}

		// Store keyboard data in array. Skip keys that do not exist.
		var keys [76]bool
		for j := 0; j < 6; j++ {
			for k := k0; k != k1; k += dk {
				l := uint(j + 6*k)
				if l == 2 || l == 4 || l == 10 || l == 41 {
					continue
				}
				keys[i] = value&(1<<l) > 0
				i++
			}
		}

		// Detect keys that were just pressed and handle each.
		for j := i - 38; j < i; j++ {
			if keys[j] && !s.keys[j] {
				layer0, lock0 := s.layer, s.lock

				// If key is defined, handle key. Otherwise, treat as 0x00 and
				// reset modifiers.
				if k := layers[s.layer][j]; k != nil {
					k.handle(s)
				} else {
					s.handleKey(0)
				}

				if layer0 == s.layer {
					// If the layer has not changed and the user is on a one
					// shot layer, return to the default layer.
					if layer0 == s.layer && s.layerOSL {
						s.layer, s.layerOSL = 0, false
					}

					// If the lock has not changed, unset it. Having this in the
					// if block ensures that we can have locks followed by keys
					// that are not on the default layer.
					if lock0 == s.lock {
						s.lock = lockNone
					}
				}
			}

			s.keys[j] = keys[j]
		}
	}
}

func handleKeyboard(f *os.File, mac string, eventChan chan event) {
	defer f.Close()

	for {
		data := make([]byte, 8)
		if _, err := io.ReadFull(f, data); err != nil {
			return
		}
		eventChan <- event{data: data, left: mac == leftMAC}
	}
}
