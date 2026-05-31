package gui

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"gitlab.com/austin-millan/otaclock/internal/core"
)

type menuScreen int

const (
	menuTools menuScreen = iota
	menuPomodoro
	menuPomodoroConfig
)

func (g *Game) menuW() int { return g.layout.Scaled(230) }

func (g *Game) menuH() int {
	if g.menuScreen == menuPomodoroConfig {
		return g.layout.Scaled(214)
	}
	return g.layout.Scaled(140)
}

func (g *Game) menuOriginX() int { return g.appW + g.layout.Scaled(8) }

func (g *Game) menuFace() text.Face {
	if g.fonts != nil {
		return g.fonts.MenuFace(14.0 * g.layout.UIScale())
	}
	return nil
}

func (g *Game) toggleMenu() {
	if g.menuVisible {
		g.hideMenu()
	} else {
		g.showMenu()
	}
}

func (g *Game) showMenu() {
	g.menuScreen = menuTools
	g.menuFocusField = 0
	g.syncPomodoroStringsFromConfig()
	g.menuVisible = true
}

func (g *Game) hideMenu() {
	g.menuVisible = false
}

func (g *Game) renderMenu(screen *ebiten.Image) {
	originX := g.menuOriginX()
	cfg := g.session.PomodoroConfig()
	status := g.session.Snapshot().Pomodoro.StatusText()
	face := g.menuFace()

	fillRect(screen, originX, 0, g.menuW(), g.menuH(), Theme.MenuBackground)

	draw := func(s string, x, y int, c color.Color) {
		drawText(screen, s, originX+x, y, face, c)
	}

	mx := g.layout.Scaled(10)
	draw("Tools", mx, g.layout.Scaled(8), Theme.Primary)

	switch g.menuScreen {
	case menuTools:
		draw("Pomodoro ▶", mx, g.layout.Scaled(36), Theme.Primary)
	case menuPomodoro:
		draw("Start", mx, g.layout.Scaled(36), Theme.Primary)
		draw("Pause", mx, g.layout.Scaled(58), Theme.Primary)
		draw("Configure ▶", mx, g.layout.Scaled(80), Theme.Primary)
		draw(status, mx, g.layout.Scaled(110), Theme.Dim)
	case menuPomodoroConfig:
		draw("Pomodoro Config", mx, g.layout.Scaled(36), Theme.Primary)
		soundLabel := "Off"
		if cfg.PlayPhaseEndSound {
			soundLabel = "On"
		}
		fields := [][2]string{
			{"Work", g.pomoWorkStr},
			{"Short", g.pomoShortStr},
			{"Long", g.pomoLongStr},
			{"Every", g.pomoEveryStr},
			{"Sound", soundLabel},
		}
		y := g.layout.Scaled(62)
		for idx, field := range fields {
			c := color.Color(Theme.Dim)
			if idx == g.menuFocusField {
				c = Theme.Primary
			}
			suffix := ": "
			if idx < 4 {
				suffix = " min: "
			}
			draw(field[0]+suffix+field[1], mx, y, c)
			y += g.layout.Scaled(24)
		}
		draw("Enter=Save  Esc=Back", mx, y+g.layout.Scaled(6), Theme.Dim)
	}
}

func (g *Game) handleMenuClick(x, y int, button ebiten.MouseButton) {
	if button != ebiten.MouseButtonLeft {
		return
	}
	switch g.menuScreen {
	case menuTools:
		if y >= g.layout.Scaled(28) {
			g.menuScreen = menuPomodoro
		}
	case menuPomodoro:
		switch {
		case y >= g.layout.Scaled(32) && y < g.layout.Scaled(54):
			g.session.PomodoroStartOrResume(g.nowMs())
		case y >= g.layout.Scaled(54) && y < g.layout.Scaled(76):
			g.session.PomodoroPause()
		case y >= g.layout.Scaled(76) && y < g.layout.Scaled(104):
			g.menuScreen = menuPomodoroConfig
			g.menuFocusField = 0
		}
	case menuPomodoroConfig:
		idx := (y - g.layout.Scaled(62)) / g.layout.Scaled(24)
		if idx >= 0 && idx < 5 {
			g.menuFocusField = idx
			if idx == 4 {
				g.session.TogglePomodoroSound()
			}
		}
	}
}

func (g *Game) handleMenuKey(key ebiten.Key) {
	if key == ebiten.KeyEscape && g.menuScreen != menuPomodoroConfig {
		g.hideMenu()
		return
	}
	switch g.menuScreen {
	case menuTools:
		if key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter {
			g.menuScreen = menuPomodoro
		}
	case menuPomodoro:
		if key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter {
			g.session.PomodoroStartOrResume(g.nowMs())
		}
	case menuPomodoroConfig:
		switch key {
		case ebiten.KeyEscape:
			g.menuScreen = menuPomodoro
		case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
			g.applyPomodoroConfigFromStrings()
			g.menuScreen = menuPomodoro
		case ebiten.KeyTab, ebiten.KeyDown:
			g.menuFocusField = (g.menuFocusField + 1) % 5
		case ebiten.KeyUp:
			g.menuFocusField = (g.menuFocusField + 4) % 5
		case ebiten.KeySpace:
			if g.menuFocusField == 4 {
				g.session.TogglePomodoroSound()
			}
		}
	}
}

func (g *Game) handleMenuTextInput(runes []rune) {
	if g.menuScreen != menuPomodoroConfig || g.menuFocusField >= 4 {
		return
	}
	digits := ""
	for _, r := range runes {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
	}
	if digits == "" {
		return
	}
	g.mutateFocusedField(func(s string) string {
		if len(s) < 3 {
			return s + digits
		}
		return s
	})
}

func (g *Game) mutateFocusedField(body func(string) string) {
	switch g.menuFocusField {
	case 0:
		g.pomoWorkStr = body(g.pomoWorkStr)
	case 1:
		g.pomoShortStr = body(g.pomoShortStr)
	case 2:
		g.pomoLongStr = body(g.pomoLongStr)
	default:
		g.pomoEveryStr = body(g.pomoEveryStr)
	}
}

func (g *Game) syncPomodoroStringsFromConfig() {
	cfg := g.session.PomodoroConfig()
	g.pomoWorkStr = strconv.Itoa(cfg.WorkMinutes)
	g.pomoShortStr = strconv.Itoa(cfg.ShortBreakMinutes)
	g.pomoLongStr = strconv.Itoa(cfg.LongBreakMinutes)
	g.pomoEveryStr = strconv.Itoa(cfg.LongBreakEvery)
}

func (g *Game) applyPomodoroConfigFromStrings() {
	current := g.session.PomodoroConfig()
	toInt := func(s string, fallback int) int {
		if v, err := strconv.Atoi(s); err == nil {
			return maxInt(1, v)
		}
		return fallback
	}
	g.session.UpdatePomodoroConfig(func(cfg *core.PomodoroConfig) {
		cfg.WorkMinutes = toInt(g.pomoWorkStr, current.WorkMinutes)
		cfg.ShortBreakMinutes = toInt(g.pomoShortStr, current.ShortBreakMinutes)
		cfg.LongBreakMinutes = toInt(g.pomoLongStr, current.LongBreakMinutes)
		cfg.LongBreakEvery = toInt(g.pomoEveryStr, current.LongBreakEvery)
	})
	g.syncPomodoroStringsFromConfig()
}
