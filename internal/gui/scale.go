package gui

// applyWidgetScale uniformly rescales the whole widget (sprite, terminal, fonts,
// window). Returns true when the scale actually changed.
func (g *Game) applyWidgetScale(newScale float64) bool {
	clamped := clampFloat(newScale, minWidgetScale, maxWidgetScale)
	if absFloat(clamped-g.widgetScale) < 0.001 {
		return false
	}

	if g.menuVisible {
		g.hideMenu()
	}

	spriteLeft := g.spriteScreenLeft()

	g.widgetScale = clamped
	g.layout = newLayout(g.widgetScale)

	g.spriteRenderer = NewSpriteRenderer(g.layout.SpriteScale())
	if g.fonts != nil {
		g.fonts.Resize(g.layout.DefaultFontNormPt(), g.layout.DefaultFontSmallPt())
	}

	g.responseH = g.layout.DefaultResponseHeight()
	g.inputH = g.layout.InputHeight()

	if g.uiMode == modeChat {
		g.appW = g.layout.ChatWindowWidth()
		g.appH = g.layout.ChatWindowHeight(g.responseH, g.inputH)
		g.responseH = maxInt(g.layout.MinResponseHeight(), g.appH-g.layout.ChatPanelY()-g.inputH)
		g.appH = g.layout.ChatWindowHeight(g.responseH, g.inputH)
	} else {
		g.appW = g.layout.WindowWidth()
		g.appH = g.layout.CompactWindowHeight()
	}

	// Keep the sprite's screen position fixed across the rescale.
	g.anchorSpriteLeft(spriteLeft)

	if g.uiMode == modeChat {
		g.recomputeInputLayout()
	}
	return true
}

// setWidgetScaleFromDrag blends the horizontal/vertical drag ratio into a scale.
func (g *Game) setWidgetScaleFromDrag(startScale float64, startW, startH, newW, newH int) bool {
	sw := float64(newW) / float64(maxInt(1, startW))
	sh := float64(newH) / float64(maxInt(1, startH))
	blended := startScale * (sw + sh) * 0.5
	return g.applyWidgetScale(blended)
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
