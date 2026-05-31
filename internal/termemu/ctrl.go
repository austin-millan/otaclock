package termemu

// CtrlC is the visible echo of ASCII ETX (0x03), as printed by terminals with
// echoctl enabled when the user presses Ctrl+C.
const CtrlC = "^C"

// VisibleCtrl renders a control character the way echoctl displays it (^A … ^Z).
func VisibleCtrl(b byte) string {
	if b >= 1 && b <= 26 {
		return "^" + string(rune('A'-1+b))
	}
	if b == 127 {
		return "^?"
	}
	return ""
}
