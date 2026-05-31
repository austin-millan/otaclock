// Package assets bundles the OtaClock sprites, fonts, and sounds directly into
// the binary so the application ships as a single self-contained executable on
// every target platform.
package assets

import (
	"embed"
	"fmt"
)

//go:embed sprites/*.png
var sprites embed.FS

//go:embed fonts/*.ttf
var fonts embed.FS

//go:embed sounds/*.wav
var sounds embed.FS

// Sprite returns the raw PNG bytes for a sprite by name (without extension),
// e.g. Sprite("otacon") or Sprite("eyes_0").
func Sprite(name string) ([]byte, error) {
	return sprites.ReadFile(fmt.Sprintf("sprites/%s.png", name))
}

// Font returns the raw TrueType bytes for a bundled MGS1 face, e.g.
// Font("MGS1_Codec").
func Font(name string) ([]byte, error) {
	return fonts.ReadFile(fmt.Sprintf("fonts/%s.ttf", name))
}

// Sound returns the raw WAV bytes for a bundled sound, e.g. Sound("exit").
func Sound(name string) ([]byte, error) {
	return sounds.ReadFile(fmt.Sprintf("sounds/%s.wav", name))
}
