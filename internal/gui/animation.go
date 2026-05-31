package gui

import (
	"math/rand"

	"gitlab.com/austin-millan/otaclock/internal/core"
)

func (g *Game) scheduleNextBlink() {
	g.nextBlinkTick = g.nowMs() + uint32(2800+rand.Intn(3701)) // 2800..6500ms
}

// updateAnimations advances blink/talking sprite frames and auto-scrolls when a
// new message arrives.
func (g *Game) updateAnimations(snap core.Snapshot) {
	now := g.nowMs()

	if g.isBlinking {
		if now-g.lastBlinkTick > 110 {
			g.isBlinking = false
			g.scheduleNextBlink()
		}
	} else if now >= g.nextBlinkTick {
		g.isBlinking = true
		g.lastBlinkTick = now
	}

	loading := snap.IsLoading
	if g.prevIsLoading && !loading {
		g.isTalking = true
		g.talkEndTick = now + 2500
	}
	g.prevIsLoading = loading
	if g.isTalking && now >= g.talkEndTick {
		g.isTalking = false
	}

	if loading || g.isTalking {
		if now-g.lastAnimTick > 250 {
			g.currentAnimFrame = (g.currentAnimFrame % 3) + 1
			g.lastAnimTick = now
		}
	} else {
		g.currentAnimFrame = 0
	}

	count := len(snap.Transcript)
	if count > g.prevMessageCount {
		g.prevMessageCount = count
		g.scrollOffset = 0
	}
}
