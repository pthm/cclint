package update

import (
	"context"
	"testing"
	"time"
)

func TestCheckWithCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := CheckWithCache(ctx)
	if err != nil {
		t.Fatalf("CheckWithCache failed: %v", err)
	}

	t.Logf("Latest: %s, Current: %s, UpdateAvailable: %v",
		info.LatestVersion, info.CurrentVersion, info.UpdateAvailable)

	if info.LatestVersion == "" {
		t.Error("LatestVersion should not be empty")
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"dev", "1.0.0", 1},
		{"1.0.0", "dev", -1},
		{"0.0.1", "0.1.1", -1},
		{"1.0.0-beta", "1.0.0", 0}, // Pre-release suffix stripped
	}

	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
