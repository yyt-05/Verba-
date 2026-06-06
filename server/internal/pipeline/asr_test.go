package pipeline

import "testing"

func TestParseTranscriptionText(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "json text payload",
			raw:  `{"text":"Hello world."}`,
			want: "Hello world.",
		},
		{
			name: "empty json text payload",
			raw:  `{"text":""}`,
			want: "",
		},
		{
			name: "plain text payload",
			raw:  `Hello world.`,
			want: "Hello world.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTranscriptionText(tt.raw); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
