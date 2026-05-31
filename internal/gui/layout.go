package gui

import (
	"image/color"
	"math"
)

// RGBA is a convenience constructor for opaque-aware colors.
func rgba(r, g, b, a uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: a} }

// Theme is the phosphor-green CODEC palette plus the MGX DATAFLOW terminal colors.
var Theme = struct {
	Primary            color.RGBA
	Dim                color.RGBA
	Border             color.RGBA
	TerminalBackground color.RGBA
	TerminalForeground color.RGBA
	TerminalDim        color.RGBA
	TerminalBorder     color.RGBA
	MenuBackground     color.RGBA
}{
	Primary:            rgba(18, 230, 55, 255),
	Dim:                rgba(12, 130, 25, 255),
	Border:             rgba(12, 88, 20, 255),
	TerminalBackground: rgba(0, 48, 0, 255),
	TerminalForeground: rgba(175, 192, 168, 255),
	TerminalDim:        rgba(120, 140, 115, 255),
	TerminalBorder:     rgba(210, 218, 205, 255),
	MenuBackground:     rgba(6, 16, 6, 245),
}

// Base design constants at widgetScale == 1.0 (the original 2x sprite / 1.5x UI layout).
const (
	baseSpriteScale        = 2
	baseUIScale            = 1.5
	chatWidthMultiplier    = 3
	chatPanelOverlapDesign = 16

	minWidgetScale     = 0.55
	maxWidgetScale     = 1.85
	widgetScaleStep    = 0.05
	resizeMarginDesign = 10

	minFontPt         = 6
	minLineHeight     = 10
	lineHeightPadding = 2
)

// LayoutMetrics derives every pixel dimension from a single uniform widget scale.
type LayoutMetrics struct {
	Scale float64
}

func newLayout(scale float64) LayoutMetrics { return LayoutMetrics{Scale: scale} }

func (m LayoutMetrics) SpriteScale() int {
	return maxInt(1, int(math.Round(float64(baseSpriteScale)*m.Scale)))
}
func (m LayoutMetrics) UIScale() float64 { return baseUIScale * m.Scale }

func (m LayoutMetrics) WindowWidth() int     { return spriteCanvasW * m.SpriteScale() }
func (m LayoutMetrics) SpriteHeight() int    { return spriteCanvasH * m.SpriteScale() }
func (m LayoutMetrics) ChatWindowWidth() int { return m.WindowWidth() * chatWidthMultiplier }

func (m LayoutMetrics) ChatPanelOverlap() int { return m.dim(chatPanelOverlapDesign) }
func (m LayoutMetrics) ChatPanelY() int       { return m.SpriteHeight() - m.ChatPanelOverlap() }

func (m LayoutMetrics) Pad() int                   { return m.dim(4) }
func (m LayoutMetrics) DefaultResponseHeight() int { return m.dim(200) }
func (m LayoutMetrics) MinResponseHeight() int     { return m.dim(150) }
func (m LayoutMetrics) InputHeight() int           { return m.dim(100) }
func (m LayoutMetrics) MaxInputHeight() int        { return m.InputHeight() * 2 }

func (m LayoutMetrics) MinChatWindowHeight() int {
	return (m.SpriteHeight() - m.ChatPanelOverlap()) + m.MinResponseHeight() + m.InputHeight()
}
func (m LayoutMetrics) MaxChatWindowHeight() int {
	return (m.SpriteHeight() - m.ChatPanelOverlap()) + m.dim(900)
}

func (m LayoutMetrics) DefaultFontNormPt() float64 {
	return math.Max(minFontPt, math.Round(11.0*m.UIScale()))
}
func (m LayoutMetrics) DefaultFontSmallPt() float64 {
	return math.Max(minFontPt, math.Round(9.0*m.UIScale()))
}
func (m LayoutMetrics) InputTextIndent() int { return m.dim(14) }
func (m LayoutMetrics) InputTextPadY() int   { return m.dim(11) }
func (m LayoutMetrics) ResizeMargin() int {
	return maxInt(4, int(math.Round(float64(resizeMarginDesign)*m.Scale)))
}

func (m LayoutMetrics) CompactWindowHeight() int { return m.SpriteHeight() }

func (m LayoutMetrics) dim(designUnits int) int {
	return maxInt(1, int(math.Round(float64(designUnits)*m.UIScale())))
}

// Scaled scales a design-unit measurement to the current UI scale.
func (m LayoutMetrics) Scaled(designUnits int) int { return m.dim(designUnits) }

func (m LayoutMetrics) ChatWindowHeight(responseH, inputH int) int {
	return m.ClampChatHeight(m.ChatPanelY() + responseH + inputH)
}

func (m LayoutMetrics) ClampChatHeight(h int) int {
	return clampInt(h, m.MinChatWindowHeight(), m.MaxChatWindowHeight())
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
