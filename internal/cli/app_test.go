package cli

import "testing"

func TestNormalizeParallel(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{input: -1, want: 1},
		{input: 0, want: 1},
		{input: 1, want: 1},
		{input: 3, want: 3},
		{input: 99, want: 10},
	}

	for _, tt := range tests {
		got := normalizeParallel(tt.input)
		if got != tt.want {
			t.Errorf("normalizeParallel(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
