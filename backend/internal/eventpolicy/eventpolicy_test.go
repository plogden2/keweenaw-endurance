package eventpolicy

import "testing"

func TestIsBluffetEventID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"full", BluffetEventIDFull, true},
		{"full uppercase", "1441674D-A011-471A-A601-722B88B117F5", true},
		{"short", BluffetEventIDShort, true},
		{"short uppercase", "B117F5", true},
		{"other uuid", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", false},
		{"other short", "a1b2c3", false},
		{"empty", "", false},
		{"whitespace only", "   ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsBluffetEventID(tc.id); got != tc.want {
				t.Errorf("IsBluffetEventID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
