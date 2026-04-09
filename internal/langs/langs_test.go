package langs

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "zh", want: "zh"},
		{in: "ZH-CN", want: "zh"},
		{in: "english", want: "en"},
		{in: "es-es", want: "es"},
		{in: "jp", want: "ja"},
	}

	for _, tt := range tests {
		got, err := Normalize(tt.in)
		if err != nil {
			t.Fatalf("Normalize(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeRejectsUnsupported(t *testing.T) {
	if _, err := Normalize("fr"); err == nil {
		t.Fatal("Normalize(fr) error = nil, want error")
	}
}
