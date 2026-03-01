package video

import "testing"

func TestGetAspectRatioLabel(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		want   string
	}{
		{"landscape 1920x1080", 1920, 1080, "16:9"},
		{"landscape 1280x720", 1280, 720, "16:9"},
		{"portrait 1080x1920", 1080, 1920, "9:16"},
		{"portrait 720x1280", 720, 1280, "9:16"},
		{"square 1080x1080", 1080, 1080, "1:1"},
		{"square 500x500", 500, 500, "1:1"},
		{"slightly wide 501x500", 501, 500, "16:9"},
		{"slightly tall 500x501", 500, 501, "9:16"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAspectRatioLabel(tt.width, tt.height)
			if got != tt.want {
				t.Errorf("getAspectRatioLabel(%d, %d) = %q, want %q", tt.width, tt.height, got, tt.want)
			}
		})
	}
}
