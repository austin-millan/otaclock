package gui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"gitlab.com/austin-millan/otaclock/assets"
	"gitlab.com/austin-millan/otaclock/internal/core"
)

const (
	spriteCanvasW = 86
	spriteCanvasH = 80
)

// Sprite-sheet offsets within the 86x80 canvas (Quartz-derived; see legacy Render.m).
const (
	posOtacon0X = 1
	posOtacon1X = 0
	posOtacon2X = -6
	posOtacon3X = -6
	posBubbleY  = 0
	posEyeX     = 60
	posEyeY     = 45

	posDateY  = posBubbleY + 4
	posDateX0 = 3
	posDateX1 = 8
	posDateX2 = 17
	posDateX3 = 22
	posDayX   = 31
	posDayY   = posBubbleY + 5

	posTimeY       = posBubbleY + 14
	posTimeX0      = 5
	posTimeX1      = 12
	posTimeX2      = 21
	posTimeX3      = 28
	posAmpmX       = 36
	posDotsX       = 18
	posClockYShift = -4

	posAlarmX             = 5
	posAlarmYQuartz       = 57
	posDotsAlarmX         = 38
	posDotsAlarmYQuartz   = 57
	posAlarmDigitsYQuartz = 57
	posAlarmX0            = 29
	posAlarmX1            = 34
	posAlarmX2            = 41
	posAlarmX3            = 46
	posSmallFontW         = 4
	posSmallFontH         = 5
)

// SpriteRenderer alpha-blits decoded PNG sprites onto a target image to draw the
// OtaClock canvas (clock, date, weekday, optional Pomodoro alarm overlay).
type SpriteRenderer struct {
	scale int

	otacon, otacon1, otacon2, otacon3 *ebiten.Image
	bubble                            *ebiten.Image
	eyes                              [5]*ebiten.Image
	dotsLarge, dotsSmall, alarm       *ebiten.Image

	sheetLarge, sheetSmall, sheetDays, sheetAmPm *ebiten.Image

	bubbleW int
	Loaded  bool
}

// NewSpriteRenderer loads all sprite assets at the given integer scale.
func NewSpriteRenderer(scale int) *SpriteRenderer {
	if scale < 1 {
		scale = 1
	}
	r := &SpriteRenderer{scale: scale, bubbleW: 56}
	r.otacon = loadSprite("otacon")
	r.otacon1 = loadSprite("otacon_1")
	r.otacon2 = loadSprite("otacon_2")
	r.otacon3 = loadSprite("otacon_3")
	r.bubble = loadSprite("bubble")
	r.dotsLarge = loadSprite("dots_large")
	r.dotsSmall = loadSprite("dots_small")
	r.alarm = loadSprite("alarm")
	for i := 0; i < 5; i++ {
		r.eyes[i] = loadSprite(fmt.Sprintf("eyes_%d", i))
	}
	r.sheetLarge = loadSprite("font_large")
	r.sheetSmall = loadSprite("font_small")
	r.sheetDays = loadSprite("font_days")
	r.sheetAmPm = loadSprite("font_time")

	if r.bubble != nil {
		r.bubbleW = r.bubble.Bounds().Dx()
	}
	r.Loaded = r.otacon != nil && r.bubble != nil
	if !r.Loaded {
		fmt.Fprintln(os.Stderr, "SpriteRenderer: core sprites missing")
	}
	return r
}

func loadSprite(name string) *ebiten.Image {
	data, err := assets.Sprite(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SpriteRenderer: missing sprite %q: %v\n", name, err)
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "SpriteRenderer: decode %q: %v\n", name, err)
		return nil
	}
	return ebiten.NewImageFromImage(img)
}

// IsPointInPomodoroOverlay hit-tests the alarm icon, countdown digits, and colon.
func (r *SpriteRenderer) IsPointInPomodoroOverlay(windowX, windowY int) bool {
	s := maxInt(1, r.scale)
	cx := windowX / s
	cy := windowY / s

	contains := func(x, y, w, h int) bool {
		return cx >= x && cx < x+w && cy >= y && cy < y+h
	}
	yFromQuartz := func(quartzY, height int) int { return spriteCanvasH - quartzY - height }

	if r.alarm != nil {
		ah := r.alarm.Bounds().Dy()
		aw := r.alarm.Bounds().Dx()
		if contains(posAlarmX, yFromQuartz(posAlarmYQuartz, ah), aw, ah) {
			return true
		}
	}

	const compress = 1
	x0 := posAlarmX0
	x3 := posAlarmX3 - compress
	yAlarm := yFromQuartz(posAlarmDigitsYQuartz, posSmallFontH)
	if contains(x0, yAlarm, (x3+posSmallFontW)-x0, posSmallFontH) {
		return true
	}

	if r.dotsSmall != nil {
		dh := r.dotsSmall.Bounds().Dy()
		dw := r.dotsSmall.Bounds().Dx()
		if contains(posDotsAlarmX, yFromQuartz(posDotsAlarmYQuartz, dh), dw, dh) {
			return true
		}
	}
	return false
}

// Render draws the full sprite canvas onto dst with its top-left at (ox, oy).
func (r *SpriteRenderer) Render(
	dst *ebiten.Image, ox, oy int,
	snap core.ClockSnapshot,
	eyeState, animState int,
	use12h, showDots bool,
	alarmDigits *[4]int, showAlarmDots bool,
) {
	s := r.scale

	blit := func(src *ebiten.Image, sx, sy int) {
		if src == nil {
			return
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(float64(s), float64(s))
		op.GeoM.Translate(float64(ox+sx*s), float64(oy+sy*s))
		dst.DrawImage(src, op)
	}

	blitSrc := func(sheet *ebiten.Image, srcX, srcY, srcW, srcH, dstX, dstY int) {
		if sheet == nil {
			return
		}
		sub, ok := sheet.SubImage(image.Rect(srcX, srcY, srcX+srcW, srcY+srcH)).(*ebiten.Image)
		if !ok {
			return
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(float64(s), float64(s))
		op.GeoM.Translate(float64(ox+dstX*s), float64(oy+dstY*s))
		dst.DrawImage(sub, op)
	}

	const lfW, lfH, lfGap = 6, 7, 1
	largeDigit := func(d, dx, dy int) { blitSrc(r.sheetLarge, d*(lfW+lfGap), 0, lfW, lfH, dx, dy) }
	const sfW, sfH, sfGap = 4, 5, 1
	smallDigit := func(d, dx, dy int) { blitSrc(r.sheetSmall, d*(sfW+sfGap), 0, sfW, sfH, dx, dy) }
	const dgW, dgH, dgGap = 17, 3, 1
	dayGlyph := func(row, dx, dy int) { blitSrc(r.sheetDays, 0, row*(dgH+dgGap), dgW, dgH, dx, dy) }
	const apW, apH, apGap = 11, 3, 1
	amPm := func(isAm bool, dx, dy int) {
		srcY := 0
		if !isAm {
			srcY = apH + apGap
		}
		blitSrc(r.sheetAmPm, 0, srcY, apW, apH, dx, dy)
	}

	blit(r.bubble, 0, posBubbleY)

	displayHour := snap.Hours
	isAm := true
	if use12h {
		h := snap.Hours % 12
		if h == 0 {
			h = 12
		}
		displayHour = h
		isAm = snap.Hours < 12
	}

	if displayHour >= 10 {
		largeDigit(displayHour/10, posTimeX0, posTimeY+posClockYShift)
	}
	largeDigit(displayHour%10, posTimeX1, posTimeY+posClockYShift)
	largeDigit(snap.Minutes/10, posTimeX2, posTimeY+posClockYShift)
	largeDigit(snap.Minutes%10, posTimeX3, posTimeY+posClockYShift)

	if use12h {
		amPm(isAm, posAmpmX, posTimeY+posClockYShift)
	}
	if showDots {
		blit(r.dotsLarge, posDotsX, posTimeY+posClockYShift)
	}

	if snap.Month/10 >= 1 {
		smallDigit(snap.Month/10, posDateX0, posDateY)
	}
	smallDigit(snap.Month%10, posDateX1, posDateY)
	smallDigit(snap.Day/10, posDateX2, posDateY)
	smallDigit(snap.Day%10, posDateX3, posDateY)

	fontRow := snap.WeekdayIndex - 1
	if snap.WeekdayIndex == 0 {
		fontRow = 6
	}
	dayGlyph(fontRow, posDayX, posDayY)

	if alarmDigits != nil {
		yFromQuartz := func(quartzY, height int) int { return spriteCanvasH - quartzY - height }
		const compress = 1
		x0, x1 := posAlarmX0, posAlarmX1
		x2, x3 := posAlarmX2-compress, posAlarmX3-compress

		if r.alarm != nil {
			blit(r.alarm, posAlarmX, yFromQuartz(posAlarmYQuartz, r.alarm.Bounds().Dy()))
		}
		if showAlarmDots && r.dotsSmall != nil {
			blit(r.dotsSmall, posDotsAlarmX, yFromQuartz(posDotsAlarmYQuartz, r.dotsSmall.Bounds().Dy()))
		}
		yAlarm := yFromQuartz(posAlarmDigitsYQuartz, posSmallFontH)
		d := alarmDigits
		if d[0] >= 0 && d[0] <= 9 {
			smallDigit(d[0], x0, yAlarm)
		}
		if d[1] >= 0 && d[1] <= 9 {
			smallDigit(d[1], x1, yAlarm)
		}
		if d[2] >= 0 && d[2] <= 9 {
			smallDigit(d[2], x2, yAlarm)
		}
		if d[3] >= 0 && d[3] <= 9 {
			smallDigit(d[3], x3, yAlarm)
		}
	}

	switch animState {
	case 1:
		blit(r.otacon1, r.bubbleW+posOtacon1X, 0)
	case 2:
		blit(r.otacon2, r.bubbleW+posOtacon2X, 0)
	case 3:
		blit(r.otacon3, r.bubbleW+posOtacon3X, 0)
	default:
		blit(r.otacon, r.bubbleW+posOtacon0X, 0)
		blit(r.eyes[clampInt(eyeState, 0, 4)], posEyeX, posEyeY)
	}
}
