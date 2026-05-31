package gui

import (
	"bytes"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomono"

	"gitlab.com/austin-millan/otaclock/assets"
)

// FontSet holds the monospace faces used by the chat terminal plus the themed
// MGS1 face used by the tools menu. All faces derive from a single embedded
// source so the binary needs no system fonts.
type FontSet struct {
	bodySource *text.GoTextFaceSource
	menuSource *text.GoTextFaceSource

	Normal *text.GoTextFace
	Small  *text.GoTextFace

	LineHeight int

	normalPt float64
	smallPt  float64
}

// NewFontSet builds the face set at the supplied point sizes.
func NewFontSet(normalPt, smallPt float64) (*FontSet, error) {
	bodySource, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		return nil, err
	}

	fs := &FontSet{bodySource: bodySource}

	if data, err := assets.Font("MGS1_Codec"); err == nil {
		if src, err := text.NewGoTextFaceSource(bytes.NewReader(data)); err == nil {
			fs.menuSource = src
		}
	}

	fs.Resize(normalPt, smallPt)
	return fs, nil
}

// Resize rebuilds the faces at new point sizes.
func (f *FontSet) Resize(normalPt, smallPt float64) {
	f.normalPt = normalPt
	f.smallPt = smallPt
	f.Normal = &text.GoTextFace{Source: f.bodySource, Size: normalPt}
	f.Small = &text.GoTextFace{Source: f.bodySource, Size: smallPt}

	if h := faceHeight(f.Small); h > 0 {
		f.LineHeight = maxInt(minLineHeight, h+lineHeightPadding)
	} else {
		f.LineHeight = 14
	}
}

// MenuFace returns the themed MGS1 face at the given size, falling back to the
// monospace body face when the MGS1 face is unavailable.
func (f *FontSet) MenuFace(size float64) *text.GoTextFace {
	if f.menuSource != nil {
		return &text.GoTextFace{Source: f.menuSource, Size: size}
	}
	return &text.GoTextFace{Source: f.bodySource, Size: size}
}
