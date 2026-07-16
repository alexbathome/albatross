package puttday

import "testing"

func TestExtractShareLink(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantOK  bool
	}{
		{
			name: "embedded in pasted message",
			content: "putt.day #65 ⛳ 6/13 Albatross · 🔥2\n" +
				"🟡🟡🟡🟡🟡🟢\n" +
				"https://putt.day/s/HRTRtJo8DO53",
			want:   "https://putt.day/s/HRTRtJo8DO53",
			wantOK: true,
		},
		{
			name:    "trailing punctuation excluded",
			content: "check this out https://putt.day/s/HRTRtJo8DO53.",
			want:    "https://putt.day/s/HRTRtJo8DO53",
			wantOK:  true,
		},
		{
			name:    "http scheme does not match",
			content: "http://putt.day/s/HRTRtJo8DO53",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "no link present",
			content: "just chatting about mini golf",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "two links present returns first",
			content: "https://putt.day/s/first https://putt.day/s/second",
			want:    "https://putt.day/s/first",
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractShareLink(tt.content)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ExtractShareLink(%q) = (%q, %v), want (%q, %v)", tt.content, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
