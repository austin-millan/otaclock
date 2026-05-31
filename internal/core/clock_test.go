package core

import "testing"

func makeSnapshot(hours, minutes, seconds, month, day, weekdayIndex int) ClockSnapshot {
	return ClockSnapshot{
		Hours:        hours,
		Minutes:      minutes,
		Seconds:      seconds,
		Month:        month,
		Day:          day,
		WeekdayIndex: weekdayIndex,
	}
}

func TestSnapshotValuesAreInValidRanges(t *testing.T) {
	snap := ClockService{}.Snapshot()
	if snap.Hours < 0 || snap.Hours >= 24 {
		t.Errorf("hours out of range: %d", snap.Hours)
	}
	if snap.Minutes < 0 || snap.Minutes >= 60 {
		t.Errorf("minutes out of range: %d", snap.Minutes)
	}
	if snap.Seconds < 0 || snap.Seconds >= 60 {
		t.Errorf("seconds out of range: %d", snap.Seconds)
	}
	if snap.WeekdayIndex < 0 || snap.WeekdayIndex >= 7 {
		t.Errorf("weekday out of range: %d", snap.WeekdayIndex)
	}
}

func TestAfternoon12hFormat(t *testing.T) {
	snap := makeSnapshot(14, 32, 0, 1, 1, 0)
	if got := snap.FormattedTime12h(); got != "2:32" {
		t.Errorf("got %q", got)
	}
	if got := snap.AMPM(); got != "PM" {
		t.Errorf("got %q", got)
	}
}

func TestMorning12hFormat(t *testing.T) {
	snap := makeSnapshot(9, 5, 0, 1, 1, 0)
	if got := snap.FormattedTime12h(); got != "9:05" {
		t.Errorf("got %q", got)
	}
	if got := snap.AMPM(); got != "AM" {
		t.Errorf("got %q", got)
	}
}

func TestMidnight12hFormat(t *testing.T) {
	snap := makeSnapshot(0, 0, 0, 1, 1, 0)
	if got := snap.FormattedTime12h(); got != "12:00" {
		t.Errorf("got %q", got)
	}
	if got := snap.AMPM(); got != "AM" {
		t.Errorf("got %q", got)
	}
}

func TestNoon12hFormat(t *testing.T) {
	snap := makeSnapshot(12, 0, 0, 1, 1, 0)
	if got := snap.FormattedTime12h(); got != "12:00" {
		t.Errorf("got %q", got)
	}
	if got := snap.AMPM(); got != "PM" {
		t.Errorf("got %q", got)
	}
}

func TestAfternoon24hFormat(t *testing.T) {
	if got := makeSnapshot(14, 32, 0, 1, 1, 0).FormattedTime24h(); got != "14:32" {
		t.Errorf("got %q", got)
	}
}

func TestMidnight24hFormat(t *testing.T) {
	if got := makeSnapshot(0, 0, 0, 1, 1, 0).FormattedTime24h(); got != "00:00" {
		t.Errorf("got %q", got)
	}
}

func TestSingleDigitHour24hFormat(t *testing.T) {
	if got := makeSnapshot(7, 8, 0, 1, 1, 0).FormattedTime24h(); got != "07:08" {
		t.Errorf("got %q", got)
	}
}

func TestDateFormatting(t *testing.T) {
	if got := makeSnapshot(0, 0, 0, 5, 27, 0).FormattedDate(); got != "05/27" {
		t.Errorf("got %q", got)
	}
}

func TestWeekdayNamesCount(t *testing.T) {
	if len(WeekdayNames) != 7 {
		t.Errorf("got %d", len(WeekdayNames))
	}
}

func TestWeekdayNamesContent(t *testing.T) {
	if WeekdayNames[0] != "Sun" || WeekdayNames[1] != "Mon" || WeekdayNames[6] != "Sat" {
		t.Error("unexpected weekday names")
	}
}

func TestWeekdayNameLookup(t *testing.T) {
	if got := makeSnapshot(0, 0, 0, 1, 1, 3).WeekdayName(); got != "Wed" {
		t.Errorf("got %q", got)
	}
}
