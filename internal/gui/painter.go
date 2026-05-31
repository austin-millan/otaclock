package gui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// fillRect paints an opaque rectangle on dst.
func fillRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

// drawText renders a single line of text with its top-left at (x, y).
func drawText(dst *ebiten.Image, s string, x, y int, face text.Face, c color.Color) int {
	if face == nil || s == "" {
		return 0
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
	return textWidth(s, face)
}

// textWidth measures the rendered width of a single line.
func textWidth(s string, face text.Face) int {
	if face == nil || s == "" {
		return 0
	}
	w, _ := text.Measure(s, face, 0)
	return int(w + 0.5)
}

// faceHeight returns the line height (ascent + descent) of a face in pixels.
func faceHeight(face text.Face) int {
	if face == nil {
		return 0
	}
	m := face.Metrics()
	return int(m.HAscent + m.HDescent + 0.5)
}

// wrapText wraps text to maxWidth, honoring explicit newlines and splitting
// over-long tokens. Mirrors SurfacePainter.wrap.
func wrapText(s string, maxWidth int, face text.Face) []string {
	if face == nil || maxWidth <= 0 {
		if s == "" {
			return []string{""}
		}
		return []string{s}
	}

	var lines []string
	paragraphs := strings.Split(s, "\n")

	for index, paragraph := range paragraphs {
		if index > 0 {
			lines = append(lines, "")
		}
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}

		current := ""
		for _, word := range strings.Split(paragraph, " ") {
			candidate := word
			if current != "" {
				candidate = current + " " + word
			}
			if textWidth(candidate, face) <= maxWidth {
				current = candidate
				continue
			}
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			parts := breakLongToken(word, maxWidth, face)
			if len(parts) == 1 {
				current = parts[0]
			} else {
				lines = append(lines, parts[:len(parts)-1]...)
				current = parts[len(parts)-1]
			}
		}
		if current != "" {
			lines = append(lines, current)
		}
	}

	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func breakLongToken(token string, maxWidth int, face text.Face) []string {
	if token == "" {
		return nil
	}
	if textWidth(token, face) <= maxWidth {
		return []string{token}
	}

	var parts []string
	var chunk strings.Builder
	for _, ch := range token {
		candidate := chunk.String() + string(ch)
		if textWidth(candidate, face) <= maxWidth {
			chunk.WriteRune(ch)
		} else {
			if chunk.Len() > 0 {
				parts = append(parts, chunk.String())
			}
			chunk.Reset()
			chunk.WriteRune(ch)
		}
	}
	if chunk.Len() > 0 {
		parts = append(parts, chunk.String())
	}
	return parts
}
