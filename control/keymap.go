package main

// #include "keycode.h"
import "C"

import (
	"log"
	"os/exec"
	"strconv"
)

type key interface {
	handle(*state)
}

type layer [76]key

// Define key type for switching layers. If a one shot layer key is pressed, the
// keyboard will reset to the default layer after the subsequent key. This
// terminology is defined by QMK in https://goo.gl/YK97JN. One shot layers do
// not need a specific key to cancel input as any undefined key will reset to
// the default layer.
type layerKey struct {
	layer int
	osl   bool
}

type switchKey string

type funcKey func(*state)

type desktopKey struct {
	colemak, qwerty uint8
}

type consumerKey uint16

type virtualKey []key

// Define type for international characters. The AltGr combination is stored in
// stdKey whereas the MacOS combination is stored in macKey. See
// https://goo.gl/RrQJnQ for all AltGr characters.
type intlKey struct {
	stdKey, macKey key
}

type unicodeKey rune

type unicodeCompoundKey []unicodeKey

type stringKey string

var (
	// Define keys for switching layers.
	kcLayer0 = layerKey{0, false}
	kcLayer1 = layerKey{1, true}
	kcLayer2 = layerKey{2, true}
	kcLayer3 = layerKey{3, false}

	// Define keys for switching devices.
	kcDeviceABC = switchKey("abc123")
	kcDeviceDEF = switchKey("def123")
	kcDeviceGHI = switchKey("ghi123")
	kcDeviceJKL = switchKey("jkl123")
	kcDeviceUIN = switchKey("uin123")
	kcDeviceMNO = switchKey("mno123")
	kcDeviceTMP = switchKey("tmp123")

	// Define keys with custom functionality.
	kcFnModLock = funcKey(func(s *state) {
		if s.lock == lockMod {
			s.lock = lockNone
		} else {
			s.lock = lockMod
		}
	})
	kcFnTmpReset = funcKey(func(s *state) {
		for {
			select {
			case <-tmpChan:
			default:
				return
			}
		}
	})
	kcFnPowerOff = keyCommand(lockPowerOff, "poweroff")
	kcFnReboot   = keyCommand(lockReboot, "reboot")

	// Define modifier keys. The left and the right modifiers are functionally
	// equivalent with the exception of left alt and right alt.
	kcLCtrl  = keyD(C.KC_LCTRL)
	kcLShift = keyD(C.KC_LSHIFT)
	kcLAlt   = keyD(C.KC_LALT)
	kcLGUI   = keyD(C.KC_LGUI)
	kcRAlt   = keyD(C.KC_RALT)

	// Define row 1 on the main layer.
	kc1, kcExclam = keyC(C.KC_1)
	kc2, kcAt     = keyC(C.KC_2)
	kc3, kcHash   = keyC(C.KC_3)
	kc4, kcDollar = keyC(C.KC_4)
	kc5, kcPrcnt  = keyC(C.KC_5)
	kc6, kcCaret  = keyC(C.KC_6)
	kc7, kcAmper  = keyC(C.KC_7)
	kc8, kcAstrsk = keyC(C.KC_8)
	kc9, kcLParen = keyC(C.KC_9)
	kc0, kcRParen = keyC(C.KC_0)

	// Define row 2 on the main layer.
	kcQLower, kcQUpper = keyC(C.KC_Q)
	kcWLower, kcWUpper = keyC(C.KC_W)
	kcFLower, kcFUpper = keyCQ(C.KC_F, C.KC_E)
	kcPLower, kcPUpper = keyCQ(C.KC_P, C.KC_R)
	kcGLower, kcGUpper = keyCQ(C.KC_G, C.KC_T)
	kcJLower, kcJUpper = keyCQ(C.KC_J, C.KC_Y)
	kcLLower, kcLUpper = keyCQ(C.KC_L, C.KC_U)
	kcULower, kcUUpper = keyCQ(C.KC_U, C.KC_I)
	kcYLower, kcYUpper = keyCQ(C.KC_Y, C.KC_O)
	kcSColon, kcColon  = keyCQ(C.KC_SCOLON, C.KC_P)

	// Define row 3 on the main layer.
	kcALower, kcAUpper = keyC(C.KC_A)
	kcRLower, kcRUpper = keyCQ(C.KC_R, C.KC_S)
	kcSLower, kcSUpper = keyCQ(C.KC_S, C.KC_D)
	kcTLower, kcTUpper = keyCQ(C.KC_T, C.KC_F)
	kcDLower, kcDUpper = keyCQ(C.KC_D, C.KC_G)
	kcHLower, kcHUpper = keyC(C.KC_H)
	kcNLower, kcNUpper = keyCQ(C.KC_N, C.KC_J)
	kcELower, kcEUpper = keyCQ(C.KC_E, C.KC_K)
	kcILower, kcIUpper = keyCQ(C.KC_I, C.KC_L)
	kcOLower, kcOUpper = keyCQ(C.KC_O, C.KC_SCOLON)

	// Define row 4 on the main layer.
	kcZLower, kcZUpper = keyC(C.KC_Z)
	kcXLower, kcXUpper = keyC(C.KC_X)
	kcCLower, kcCUpper = keyC(C.KC_C)
	kcVLower, kcVUpper = keyC(C.KC_V)
	kcBLower, kcBUpper = keyC(C.KC_B)
	kcKLower, kcKUpper = keyCQ(C.KC_K, C.KC_N)
	kcMLower, kcMUpper = keyC(C.KC_M)
	kcComma, kcLAngle  = keyC(C.KC_COMMA)
	kcDot, kcRAngle    = keyC(C.KC_DOT)
	kcSlash, kcQues    = keyC(C.KC_SLASH)

	// Define tab, space, and enter.
	kcTab   = keyD(C.KC_TAB)
	kcSpace = keyD(C.KC_SPACE)
	kcEnter = keyD(C.KC_ENTER)

	// Define additional characters in the printable ASCII range.
	kcBslash, kcPipe   = keyC(C.KC_BSLASH)
	kcGrave, kcTilde   = keyC(C.KC_GRAVE)
	kcMinus, kcUnders  = keyC(C.KC_MINUS)
	kcEqual, kcPlus    = keyC(C.KC_EQUAL)
	kcLBrack, kcLBrace = keyC(C.KC_LBRACKET)
	kcRBrack, kcRBrace = keyC(C.KC_RBRACKET)
	kcQuote, kcDQuote  = keyC(C.KC_QUOTE)

	// Define additional keys in the top 5 rows on the main layer.
	kcPrtScn = keyD(C.KC_PSCREEN)
	kcApp    = keyD(C.KC_APPLICATION)
	kcCapsLk = keyD(C.KC_CAPSLOCK)
	kcVolMut = consumerKey(0xe2)
	kcEscape = keyD(C.KC_ESCAPE)
	kcBspace = keyD(C.KC_BSPACE)
	kcVolDn  = consumerKey(0xea)
	kcVolUp  = consumerKey(0xe9)
	kcLeft   = keyD(C.KC_LEFT)
	kcDown   = keyD(C.KC_DOWN)
	kcUp     = keyD(C.KC_UP)
	kcRight  = keyD(C.KC_RIGHT)

	// Define additional keys for the left and the right thumb clusters. The
	// Home and End keys are not supported on MacOS and we must instead use the
	// GUI key with an arrow key.
	kcDelete = keyD(C.KC_DELETE)
	kcHome   = intlKey{stdKey: keyD(C.KC_HOME), macKey: virtualKey{kcLGUI, kcLeft}}
	kcPageUp = keyD(C.KC_PGUP)
	kcEnd    = intlKey{stdKey: keyD(C.KC_END), macKey: virtualKey{kcLGUI, kcRight}}
	kcInsert = keyD(C.KC_INSERT)
	kcPageDn = keyD(C.KC_PGDOWN)

	// Define shortcuts for Firefox.
	kcFFMenu = virtualKey{kcLCtrl, kcLLower, kcLShift, kcTab, kcLShift, kcTab, kcApp}

	// Define Spanish accented vowels.
	kcATilde = keyESTilde(kcALower)
	kcETilde = keyESTilde(kcELower)
	kcITilde = keyESTilde(kcILower)
	kcOTilde = keyESTilde(kcOLower)
	kcUTilde = keyESTilde(kcULower)

	// Define additional Spanish characters.
	kcNTilde = intlKey{
		stdKey: virtualKey{kcRAlt, kcNLower},
		macKey: virtualKey{kcLAlt, kcNLower, kcNLower},
	}
	kcUDier = intlKey{
		stdKey: virtualKey{kcRAlt, kcYLower},
		macKey: virtualKey{kcLAlt, kcULower, kcULower},
	}
	kcExclInv = intlKey{
		stdKey: virtualKey{kcRAlt, kcLShift, kc1},
		macKey: virtualKey{kcLAlt, kc1},
	}
	kcQuesInv = intlKey{
		stdKey: virtualKey{kcRAlt, kcSlash},
		macKey: virtualKey{kcLAlt, kcLShift, kcSlash},
	}

	// Define emojis.
	kcEmGrin  = unicodeKey(0x1f604)
	kcEmCry   = unicodeKey(0x1f622)
	kcEmJoy   = unicodeKey(0x1f602)
	kcEmSweat = unicodeKey(0x1f605)
	kcEmSmile = unicodeKey(0x1f600)
	kcEmThumb = unicodeKey(0x1f44d)
	kcEmThink = unicodeKey(0x1f914)
	kcEmWink  = unicodeKey(0x1f609)
	kcEmPout  = unicodeKey(0x1f621)
	kcEmFPalm = unicodeCompoundKey{0x1f926, 0x2642}
)

var asciiTable ['~' + 1 - ' ']key = [...]key{
	kcSpace, kcExclam, kcDQuote, kcHash, kcDollar, kcPrcnt, kcAmper, kcQuote,
	kcLParen, kcRParen, kcAstrsk, kcPlus, kcComma, kcMinus, kcDot, kcSlash,
	kc0, kc1, kc2, kc3, kc4, kc5, kc6, kc7,
	kc8, kc9, kcColon, kcSColon, kcLAngle, kcEqual, kcRAngle, kcQues,
	kcAt, kcAUpper, kcBUpper, kcCUpper, kcDUpper, kcEUpper, kcFUpper, kcGUpper,
	kcHUpper, kcIUpper, kcJUpper, kcKUpper, kcLUpper, kcMUpper, kcNUpper, kcOUpper,
	kcPUpper, kcQUpper, kcRUpper, kcSUpper, kcTUpper, kcUUpper, kcVUpper, kcWUpper,
	kcXUpper, kcYUpper, kcZUpper, kcLBrack, kcBslash, kcRBrack, kcCaret, kcUnders,
	kcGrave, kcALower, kcBLower, kcCLower, kcDLower, kcELower, kcFLower, kcGLower,
	kcHLower, kcILower, kcJLower, kcKLower, kcLLower, kcMLower, kcNLower, kcOLower,
	kcPLower, kcQLower, kcRLower, kcSLower, kcTLower, kcULower, kcVLower, kcWLower,
	kcXLower, kcYLower, kcZLower, kcLBrace, kcPipe, kcRBrace, kcTilde,
}

var layers []layer = []layer{
	// Define main layer.
	{
		nil, kc1, kc2, kc3, kc4, kc5, kcPrtScn,
		kcTab, kcQLower, kcWLower, kcFLower, kcPLower, kcGLower, kcCapsLk,
		kcEscape, kcALower, kcRLower, kcSLower, kcTLower, kcDLower,
		kcLayer2, kcZLower, kcXLower, kcCLower, kcVLower, kcBLower, kcVolDn,
		nil, nil, kcLAlt, kcLeft, kcDown,
		kcLGUI, kcPageUp, kcLCtrl, kcSpace, kcHome, kcDelete,
		kcFFMenu, kc6, kc7, kc8, kc9, kc0, kcApp,
		kcVolMut, kcJLower, kcLLower, kcULower, kcYLower, kcSColon, kcBslash,
		kcHLower, kcNLower, kcELower, kcILower, kcOLower, kcBspace,
		kcVolUp, kcKLower, kcMLower, kcComma, kcDot, kcSlash, kcLayer2,
		kcUp, kcRight, kcLAlt, kcFnModLock, kcLayer3,
		kcInsert, kcEnd, kcEnter, kcLShift, kcPageDn, kcLayer1,
	},

	// Define layer for F1-F24 and keys for switching between devices. The
	// function keys are not given their own variable since they are simple to
	// define and not need elsewhere.
	{
		nil, kcDeviceABC, kcDeviceDEF, kcDeviceGHI, kcDeviceJKL, kcDeviceUIN, nil,
		nil, keyD(C.KC_F1), keyD(C.KC_F2), keyD(C.KC_F3), keyD(C.KC_F4), nil, nil,
		nil, keyD(C.KC_F5), keyD(C.KC_F6), keyD(C.KC_F7), keyD(C.KC_F8), nil,
		nil, keyD(C.KC_F9), keyD(C.KC_F10), keyD(C.KC_F11), keyD(C.KC_F12), nil, nil,
		nil, nil, nil, nil, nil,
		nil, nil, kcFnTmpReset, kcDeviceTMP, nil, nil,
		nil, kcDeviceMNO, nil, nil, nil, nil, nil,
		nil, nil, keyD(C.KC_F13), keyD(C.KC_F14), keyD(C.KC_F15), keyD(C.KC_F16), nil,
		nil, keyD(C.KC_F17), keyD(C.KC_F18), keyD(C.KC_F19), keyD(C.KC_F20), nil,
		nil, nil, keyD(C.KC_F21), keyD(C.KC_F22), keyD(C.KC_F23), keyD(C.KC_F24), nil,
		nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil,
	},

	// Define layer for additional characters.
	{
		kcFnPowerOff, kcEmGrin, kcEmCry, kcEmJoy, kcEmSweat, kcEmSmile, nil,
		nil, nil, kcExclInv, kcTilde, kcGrave, kcHash, nil,
		nil, kcATilde, kcExclam, kcPlus, kcEqual, kcDollar,
		nil, kcQuesInv, kcAt, kcLBrack, kcRBrack, kcPrcnt, nil,
		nil, nil, nil, nil, nil,
		nil, nil, kcDQuote, kcQuote, nil, nil,
		nil, kcEmThumb, kcEmThink, kcEmWink, kcEmPout, kcEmFPalm, kcFnReboot,
		nil, kcCaret, kcUDier, kcUTilde, nil, nil, nil,
		kcAmper, kcNTilde, kcETilde, kcITilde, kcOTilde, nil,
		nil, kcAstrsk, kcMinus, kcUnders, kcLBrace, kcRBrace, nil,
		nil, nil, nil, nil, nil,
		nil, nil, kcRParen, kcLParen, nil, nil,
	},

	// Define number pad layer.
	{
		nil, nil, nil, nil, nil, nil, nil,
		kcTab, nil, nil, nil, nil, nil, nil,
		nil, kcSlash, kcAstrsk, kcMinus, kcPlus, nil,
		nil, nil, nil, kcEqual, kcDot, nil, nil,
		nil, nil, nil, nil, nil,
		nil, nil, kcLCtrl, kcSpace, nil, nil,
		nil, nil, nil, nil, nil, nil, nil,
		nil, nil, kc7, kc8, kc9, kcColon, nil,
		nil, kc4, kc5, kc6, kc0, kcBspace,
		nil, nil, kc1, kc2, kc3, nil, nil,
		nil, nil, nil, nil, kcLayer0,
		nil, nil, kcEnter, kcLShift, nil, nil,
	},
}

func (k layerKey) handle(s *state) {
	s.layer, s.layerOSL = k.layer, k.osl
}

func (k switchKey) handle(s *state) {
	// Reset all active modifiers.
	for key, value := range s.modifiers {
		if value {
			s.releaseModifier(key)
		}
	}

	// Update device.
	s.device = string(k)
}

func (k funcKey) handle(s *state) {
	k(s)
}

func (k desktopKey) handle(s *state) {
	if devices[s.device].qwerty && k.qwerty != 0 {
		s.handleKey(k.qwerty)
	} else {
		s.handleKey(k.colemak)
	}
}

func (k consumerKey) handle(s *state) {
	// Send using consumer report. Send 0x00 key to reset modifiers if
	// appropriate.
	devices[s.device].writer.sendConsumer(uint16(k))
	devices[s.device].writer.sendConsumer(0)
	s.handleKey(0)
}

func (k virtualKey) handle(s *state) {
	for _, value := range k {
		value.handle(s)
	}
}

func (k intlKey) handle(s *state) {
	// This expects Android, Linux, and Windows to have an international
	// keyboard with AltGr support. The default layout is fine on MacOS. There
	// is no default cause since the cases are exhaustive.
	switch devices[s.device].platform {
	case platAndroid, platLinux, platWindows:
		k.stdKey.handle(s)
	case platMacOS:
		k.macKey.handle(s)
	}
}

func (k unicodeKey) handle(s *state) {
	// Input Unicode character with Ctrl-Shuft-U on Linux. On other platforms,
	// treat as no-op and reset modifiers.
	if devices[s.device].platform == platLinux {
		virtualKey{kcLCtrl, kcLShift, kcULower}.handle(s)
		stringKey(strconv.FormatInt(int64(k), 16)).handle(s)
		kcSpace.handle(s)
	} else {
		s.handleKey(0)
	}
}

func (k unicodeCompoundKey) handle(s *state) {
	zwj := unicodeKey(0x200d)
	for key, value := range k {
		if key != 0 {
			zwj.handle(s)
		}
		value.handle(s)
	}
}

func (k stringKey) handle(s *state) {
	for _, value := range k {
		switch {
		case value >= ' ' && value <= '~':
			asciiTable[value-' '].handle(s)
		case value == '\n':
			kcEnter.handle(s)
		case value == '\t':
			kcTab.handle(s)
		}
	}
}

func keyCommand(l lock, cmd string, args ...string) funcKey {
	return funcKey(func(s *state) {
		if s.lock == l {
			if err := exec.Command(cmd, args...).Run(); err != nil {
				log.Fatal(err)
			}
		} else {
			s.lock = l
		}
	})
}

func keyD(code uint8) desktopKey {
	return desktopKey{colemak: code}
}

func keyC(colemak uint8) (desktopKey, virtualKey) {
	return keyCQ(colemak, 0)
}

func keyCQ(colemak, qwerty uint8) (desktopKey, virtualKey) {
	base := desktopKey{colemak: colemak, qwerty: qwerty}
	shifted := virtualKey{kcLShift, base}
	return base, shifted
}

func keyESTilde(vowel key) intlKey {
	return intlKey{
		stdKey: virtualKey{kcRAlt, vowel},
		macKey: virtualKey{kcLAlt, kcELower, vowel},
	}
}
