package common

import (
	"time"
)

func Now() time.Time {
	return time.Now().UTC()
}

func UnixMillis(t time.Time) int64 {
	return t.UnixMilli()
}

func StartOfWindow(now time.Time, window time.Duration) time.Time {
	return now.Add(-window)
}

func TimeOrNow(t time.Time) time.Time {
	if t.IsZero() {
		return Now()
	}
	return t.UTC()
}
