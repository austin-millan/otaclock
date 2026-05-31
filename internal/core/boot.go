package core

import "strings"

// BootLoadingPrefix is the animated header shown before the kernel log scrolls.
const BootLoadingPrefix = "Loading MGX"

// BootLine is one line of the Policenauts / MONORITH MGX boot transcript.
type BootLine struct {
	Text string
	// DelayAfterMs is the pause after this line finishes before the next begins.
	DelayAfterMs uint32
	// CharMs, when > 0, types the line in character-by-character at this interval.
	CharMs uint32
}

func bl(text string, delay uint32) BootLine {
	return BootLine{Text: text, DelayAfterMs: delay}
}

// BootLines is the MGX kernel boot transcript, printed in order after the
// animated "Loading MGX........" pause.
var BootLines = []BootLine{
	bl("Memory: 1048576k/1049216k available (1162k kernel code, 6812k", 90),
	bl("482k data,240k init, Ok highmem)", 140),
	bl("Dentry-cache hash table entries: 32768 (order: 6, 262144 bytes)", 70),
	bl("Buffer-cache hash table entries: 8192 (order: 3, 32768 bytes)", 70),
	bl("Page-cache hash table entries: 65536 (order: 6, 262144 bytes)", 70),
	bl("Inode-cache hash table entries: 16384 (order: 5, 131072 bytes)", 220),
	bl("Enabling fast FPU save and restore... done.", 160),
	bl("Enabling unmasked SIMD FPU exception support... done.", 160),
	bl("Checking 'hlt' instruction... OK.", 480),
	bl("MGX NET9.0 for MGX 6.0", 320),
	bl("BIOS Vendor: Konami Computer Entertainment Japan", 520),
	bl("BIOS Version: 5.73", 380),
	bl("Starting kswapd v3.0", 120),
	bl("pty: 256 Unix98 ptys configured", 100),
	bl("Uniform Multi-Platform E-IDE driver Revision: 9.11", 90),
	bl("DMA at 0xf000-0xf007, BIOS settings: da0:DMA, da1:pio", 90),
	bl("19623/16/255 from BIOS ignored", 100),
	bl("da0: TOKUGAWA HD4192 E, ATA DISK drive", 85),
	bl("da1: TOKUGAWA CF2048J, REMOVABLE drive", 85),
	bl("ide0 at 0x1f0-0x1f7, 0x3f6 on irq 14", 90),
	bl("158856/16/63, UDMA(33)", 650),
	bl("Partition check:", 420),
	bl("eth0: ARMSTECH Corporation 826557, 00:03:47:70:29:86, IRQ 9.", 110),
	bl("NET9: MGX TCP/IP 4.0 for NET9.0", 140),
	bl("IP Proyocols: ICMP, UDP, TCP, IGMP", 100),
	bl("IP: routing cache hash table of 512 buckets, 4Kbytes", 90),
	bl("ICP: hash tables configured (established 8192 bind 8192)", 90),
	bl("x domain sockets 1.0/SMP for MGX NET9.0", 380),
	bl("VFS: Mounted root (ext4 filesystem) readonly.", 200),
	bl("Freeing unused kernel memory: 176k freed", 280),
	bl("Adding Swap: 211640k swap space (priority -1)", 350),
	bl("INIT: Version 5.64", 400),
	bl("Welcome to MONORITH SERVICE SYSTEM", 650),
	bl("PROVIDED BY STRATELOGIC INC.,LTD.", 500),
	bl("######################CAUTION#########################", 900),
	bl("FOR MILITARY USE ONLY.", 750),
	bl("UNAUTHORIZED PERSONNEL MAKING ACCESS OR ATTEMPTING ACCESS WILL BE", 650),
	bl("PENALIZED SEVERELY UNDER MARTIAL LAW.", 1100),
	bl("Mounting proc filesystem", 180),
	bl("Configuring kernel parameters", 160),
	bl("Loading default keymap", 140),
	bl("Activating swap partitions", 160),
	bl("Setting hostname policenauts", 200),
	bl("Checking root filesystem", 380),
	bl("H9/dev/da01: clean, 2676/788800 files, 492275/3155200  blocks", 240),
	bl("Remounting root filesystem in read-write mode", 200),
	bl("Finding module dependencies", 180),
	bl("/dev/da05: clean, 21/82328 files, 10415/329301 blocks", 220),
	bl("/dev/da06: clean, 75098/9729792 files, 316284/38919168 blocks", 260),
	bl("/dev/da07: clean, 20881/9729792 files, 834953/38919168 blocks", 300),
	bl("Mounting local filesystems", 220),
	bl("Enabling swap space", 200),
	bl("INIT: Entering runlevel: 4", 450),
	bl("Entering non-interactive startup", 280),
	bl("Setting network parameters", 200),
	bl("Bringing up interface lo", 160),
	bl("Bringing up interface eth0", 320),
	bl("MONORITH MGX 2.1.5", 400),
	bl("Kernel 6.2.38", 1400),
}

const (
	bootLoadingDotIntervalMs uint32 = 320
	bootLoadingDotCount             = 8
	bootPauseAfterLoadingMs  uint32 = 1100
)

// BootFrame is the rendered state of the boot playback at a given tick.
type BootFrame struct {
	Lines           []string
	ShowBlockCursor bool
}

// BootPlayback drives the timed reveal of the MGX boot sequence. It is pure
// timing logic: callers supply a monotonic millisecond tick.
type BootPlayback struct {
	startedAtMs uint32
	started     bool
}

// HasStarted reports whether playback has begun.
func (b *BootPlayback) HasStarted() bool { return b.started }

// StartIfNeeded begins the sequence the first time it is requested.
func (b *BootPlayback) StartIfNeeded(tickMs uint32) {
	if b.started {
		return
	}
	b.startedAtMs = tickMs
	b.started = true
}

// Restart replays the sequence from the beginning.
func (b *BootPlayback) Restart(tickMs uint32) {
	b.startedAtMs = tickMs
	b.started = true
}

func bootLoadingDoneMs() uint32 {
	return uint32(bootLoadingDotCount)*bootLoadingDotIntervalMs + bootPauseAfterLoadingMs
}

// IsComplete reports whether the entire sequence has finished printing.
func (b *BootPlayback) IsComplete(tickMs uint32) bool {
	if !b.started {
		return false
	}
	elapsed := tickMs - b.startedAtMs
	if elapsed < bootLoadingDoneMs() {
		return false
	}
	_, complete := buildBootLines(elapsed - bootLoadingDoneMs())
	return complete
}

// Frame returns the visible boot output at the supplied tick.
func (b *BootPlayback) Frame(tickMs uint32) BootFrame {
	if !b.started {
		return BootFrame{}
	}

	elapsed := tickMs - b.startedAtMs
	if elapsed < bootLoadingDoneMs() {
		dotPhase := elapsed / bootLoadingDotIntervalMs
		dots := int(dotPhase)
		if dots > bootLoadingDotCount {
			dots = bootLoadingDotCount
		}
		loading := BootLoadingPrefix + strings.Repeat(".", dots)
		holdCursor := dots >= bootLoadingDotCount &&
			(elapsed-uint32(dots)*bootLoadingDotIntervalMs) > bootPauseAfterLoadingMs/2
		return BootFrame{Lines: []string{loading}, ShowBlockCursor: holdCursor}
	}

	lines, complete := buildBootLines(elapsed - bootLoadingDoneMs())
	cursorBlink := complete && (tickMs/480)%2 == 0
	return BootFrame{Lines: lines, ShowBlockCursor: cursorBlink}
}

func buildBootLines(bootElapsed uint32) (lines []string, complete bool) {
	out := []string{BootLoadingPrefix + strings.Repeat(".", bootLoadingDotCount)}

	var schedule uint32
	for _, entry := range BootLines {
		if bootElapsed < schedule {
			return out, false
		}

		local := bootElapsed - schedule
		out = append(out, visibleText(entry, local))

		var typingDuration uint32
		if entry.CharMs > 0 {
			typingDuration = uint32(len([]rune(entry.Text))) * entry.CharMs
		}
		lineDuration := typingDuration
		if lineDuration < 1 {
			lineDuration = 1
		}
		if local < lineDuration {
			return out, false
		}

		schedule += lineDuration
		schedule += entry.DelayAfterMs
	}

	return out, true
}

func visibleText(entry BootLine, elapsedInLineMs uint32) string {
	if entry.CharMs == 0 || entry.Text == "" {
		return entry.Text
	}
	runes := []rune(entry.Text)
	chars := int(elapsedInLineMs/entry.CharMs) + 1
	if chars < 1 {
		chars = 1
	}
	if chars > len(runes) {
		chars = len(runes)
	}
	return string(runes[:chars])
}
