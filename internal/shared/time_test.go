package shared

import (
	"testing"
	"time"
)

func TestTimeAgo(t *testing.T) {
	fixed := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	orig := TimeAgoNow
	TimeAgoNow = func() time.Time { return fixed }
	defer func() { TimeAgoNow = orig }()

	tests := []struct {
		name   string
		offset time.Duration
		want   string
	}{
		{name: "0s", offset: 0, want: "just now"},
		{name: "30s", offset: 30 * time.Second, want: "just now"},
		{name: "1m", offset: time.Minute, want: "1 minute ago"},
		{name: "5m", offset: 5 * time.Minute, want: "5 minutes ago"},
		{name: "1h", offset: time.Hour, want: "1 hour ago"},
		{name: "6h", offset: 6 * time.Hour, want: "6 hours ago"},
		{name: "1d", offset: 24 * time.Hour, want: "1 day ago"},
		{name: "3d", offset: 72 * time.Hour, want: "3 days ago"},
		{name: "10d", offset: 240 * time.Hour, want: "Jun 5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TimeAgo(fixed.Add(-tc.offset))
			if got != tc.want {
				t.Errorf("TimeAgo() = %q, want %q", got, tc.want)
			}
		})
	}
}
