package executor

import (
	"fmt"
	"forge/pkg/browser"
	"forge/pkg/uia"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32               = syscall.MustLoadDLL("user32.dll")
	procSendInput        = user32.MustFindProc("SendInput")
	procGetSystemMetrics = user32.MustFindProc("GetSystemMetrics")
)

type Action struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"` // Added for click_element
	X    int    `json:"x,omitempty"`
	Y    int    `json:"y,omitempty"`
	Text string `json:"text,omitempty"`
	Ms   int    `json:"ms,omitempty"`
	Key  string `json:"key,omitempty"`
}

func ExecutePlan(actions []Action) {
	for _, act := range actions {
		switch act.Type {
		case "move":
			moveMouse(act.X, act.Y)
		case "click":
			clickMouse()
		case "click_element", "click_text":
			target := act.Name
			if act.Type == "click_text" {
				target = act.Text
			}
			el, err := uia.WaitForElement(target, 5000)
			if err == nil && el != nil {
				fmt.Printf("Found %s at (%d, %d), clicking...\n", target, el.X, el.Y)
				moveMouse(el.X, el.Y)
				time.Sleep(100 * time.Millisecond)
				clickMouse()
			} else {
				fmt.Printf("Failed to find %s on screen: %v\n", target, err)
			}
		case "type":
			typeText(act.Text)
		case "key":
			pressSpecialKey(act.Key)
		case "sleep":
			time.Sleep(time.Duration(act.Ms) * time.Millisecond)
		case "browser_navigate":
			browser.Navigate(act.Text)
		case "browser_click_dom":
			browser.Click(act.Name) // Using Name as the CSS selector
		case "browser_type_dom":
			browser.Type(act.Name, act.Text) // Using Name as CSS selector, Text as text to type
		}
		time.Sleep(100 * time.Millisecond) // small delay between actions
	}
}

func moveMouse(x, y int) {
	type MOUSEINPUT struct {
		Dx          int32
		Dy          int32
		MouseData   uint32
		DwFlags     uint32
		Time        uint32
		DwExtraInfo uintptr
	}
	type INPUT struct {
		Type uint32
		Mi   MOUSEINPUT
		Pad  [8]byte
	}

	w, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	h, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	screenW := int32(w)
	screenH := int32(h)
	
	if screenW == 0 { screenW = 1920 }
	if screenH == 0 { screenH = 1080 }

	absX := int32(x) * 65535 / screenW
	absY := int32(y) * 65535 / screenH

	var i INPUT
	i.Type = 0
	i.Mi.Dx = absX
	i.Mi.Dy = absY
	i.Mi.DwFlags = 0x8000 | 0x0001 // MOUSEEVENTF_ABSOLUTE | MOUSEEVENTF_MOVE

	procSendInput.Call(1, uintptr(unsafe.Pointer(&i)), uintptr(unsafe.Sizeof(i)))
}

func clickMouse() {
	type MOUSEINPUT struct {
		Dx          int32
		Dy          int32
		MouseData   uint32
		DwFlags     uint32
		Time        uint32
		DwExtraInfo uintptr
	}
	type INPUT struct {
		Type uint32
		Mi   MOUSEINPUT
		Pad  [8]byte
	}

	var down INPUT
	down.Type = 0
	down.Mi.DwFlags = 0x0002 // MOUSEEVENTF_LEFTDOWN

	var up INPUT
	up.Type = 0
	up.Mi.DwFlags = 0x0004 // MOUSEEVENTF_LEFTUP

	inputs := []INPUT{down, up}
	procSendInput.Call(2, uintptr(unsafe.Pointer(&inputs[0])), uintptr(unsafe.Sizeof(down)))
}

func typeText(text string) {
	type KEYBDINPUT struct {
		WVk         uint16
		WScan       uint16
		DwFlags     uint32
		Time        uint32
		DwExtraInfo uintptr
	}
	type INPUT struct {
		Type uint32
		Ki   KEYBDINPUT
		Pad  [8]byte
	}

	var inputs []INPUT
	for _, char := range text {
		// key down
		var down INPUT
		down.Type = 1
		down.Ki.WScan = uint16(char)
		down.Ki.DwFlags = 0x0004 // KEYEVENTF_UNICODE

		// key up
		var up INPUT
		up.Type = 1
		up.Ki.WScan = uint16(char)
		up.Ki.DwFlags = 0x0004 | 0x0002 // KEYEVENTF_UNICODE | KEYEVENTF_KEYUP

		inputs = append(inputs, down, up)
	}

	if len(inputs) > 0 {
		procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), uintptr(unsafe.Sizeof(inputs[0])))
	}
}

func pressSpecialKey(keyName string) {
	type KEYBDINPUT struct {
		WVk         uint16
		WScan       uint16
		DwFlags     uint32
		Time        uint32
		DwExtraInfo uintptr
	}
	type INPUT struct {
		Type uint32
		Ki   KEYBDINPUT
		Pad  [8]byte
	}

	var inputs []INPUT

	addKey := func(vk uint16) {
		var down, up INPUT
		down.Type, up.Type = 1, 1
		down.Ki.WVk, up.Ki.WVk = vk, vk
		up.Ki.DwFlags = 0x0002 // KEYEVENTF_KEYUP
		inputs = append(inputs, down, up)
	}

	addCombo := func(mods []uint16, key uint16) {
		for _, m := range mods {
			var down INPUT
			down.Type = 1
			down.Ki.WVk = m
			inputs = append(inputs, down)
		}
		addKey(key)
		for i := len(mods) - 1; i >= 0; i-- {
			var up INPUT
			up.Type = 1
			up.Ki.WVk = mods[i]
			up.Ki.DwFlags = 0x0002
			inputs = append(inputs, up)
		}
	}

	switch keyName {
	case "win": addKey(0x5B)
	case "enter": addKey(0x0D)
	case "tab": addKey(0x09)
	case "playpause": addKey(0xB3) // VK_MEDIA_PLAY_PAUSE
	case "audio_next": addKey(0xB0) // VK_MEDIA_NEXT_TRACK
	case "audio_prev": addKey(0xB1) // VK_MEDIA_PREV_TRACK
	case "audio_vol_up": addKey(0xAF) // VK_VOLUME_UP
	case "audio_vol_down": addKey(0xAE) // VK_VOLUME_DOWN
	case "audio_mute": addKey(0xAD) // VK_VOLUME_MUTE
	case "win+r": addCombo([]uint16{0x5B}, 0x52) // VK_LWIN + R
	case "ctrl+n": addCombo([]uint16{0x11}, 0x4E) // VK_CONTROL + N
	case "ctrl+p": addCombo([]uint16{0x11}, 0x50) // VK_CONTROL + P
	case "ctrl+s": addCombo([]uint16{0x11}, 0x53) // VK_CONTROL + S
	case "ctrl+shift+s": addCombo([]uint16{0x11, 0x10}, 0x53) // VK_CONTROL + VK_SHIFT + S
	case "ctrl+f": addCombo([]uint16{0x11}, 0x46) // VK_CONTROL + F
	case "ctrl+h": addCombo([]uint16{0x11}, 0x48) // VK_CONTROL + H
	case "ctrl+k": addCombo([]uint16{0x11}, 0x4B) // VK_CONTROL + K
	default: return
	}

	if len(inputs) > 0 {
		procSendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), uintptr(unsafe.Sizeof(inputs[0])))
	}
}
