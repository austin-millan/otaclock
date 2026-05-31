package core

import (
	"fmt"
	"time"
)

// WeekdayNames maps a zero-based weekday index (0 = Sunday) to its short label.
var WeekdayNames = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// ClockSnapshot is an immutable view of the wall clock at a single instant.
type ClockSnapshot struct {
	Hours        int
	Minutes      int
	Seconds      int
	Month        int
	Day          int
	WeekdayIndex int // 0 = Sunday ... 6 = Saturday
}

// FormattedTime12h renders the hour:minute pair in 12-hour form (no leading zero on the hour).
func (c ClockSnapshot) FormattedTime12h() string {
	h := c.Hours
	switch {
	case h == 0:
		h = 12
	case h > 12:
		h -= 12
	}
	return fmt.Sprintf("%d:%02d", h, c.Minutes)
}

// FormattedTime24h renders the zero-padded 24-hour hour:minute pair.
func (c ClockSnapshot) FormattedTime24h() string {
	return fmt.Sprintf("%02d:%02d", c.Hours, c.Minutes)
}

// FormattedTimeWithSeconds24h renders hour:minute:second in 24-hour form.
func (c ClockSnapshot) FormattedTimeWithSeconds24h() string {
	return fmt.Sprintf("%02d:%02d:%02d", c.Hours, c.Minutes, c.Seconds)
}

// AMPM returns "AM" before noon and "PM" otherwise.
func (c ClockSnapshot) AMPM() string {
	if c.Hours < 12 {
		return "AM"
	}
	return "PM"
}

// FormattedDate renders the zero-padded month/day pair.
func (c ClockSnapshot) FormattedDate() string {
	return fmt.Sprintf("%02d/%02d", c.Month, c.Day)
}

// WeekdayName returns the short label for the snapshot's weekday.
func (c ClockSnapshot) WeekdayName() string {
	return WeekdayNames[((c.WeekdayIndex%7)+7)%7]
}

// ClockService produces a ClockSnapshot from the system clock on demand.
type ClockService struct{}

// SnapshotAt reads the supplied instant into a ClockSnapshot.
func (ClockService) SnapshotAt(now time.Time) ClockSnapshot {
	return ClockSnapshot{
		Hours:        now.Hour(),
		Minutes:      now.Minute(),
		Seconds:      now.Second(),
		Month:        int(now.Month()),
		Day:          now.Day(),
		WeekdayIndex: int(now.Weekday()), // time.Weekday: Sunday == 0
	}
}

// Snapshot reads the current local time into a ClockSnapshot.
func (s ClockService) Snapshot() ClockSnapshot {
	return s.SnapshotAt(time.Now())
}
