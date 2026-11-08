package meeting

import (
	"slices"
	"testing"
)

func TestInsertLandsOnlyInGapsBetweenScoutsNotYetCheckedIn(t *testing.T) {
	t.Parallel()
	// a and c are checked in; b, d, e are not. Gaps: before b, before d,
	// before e, after e.
	order := []string{"a", "b", "c", "d", "e"}
	checked := map[string]string{"a": "x", "c": "y"}
	want := [][]string{
		{"a", "new", "b", "c", "d", "e"},
		{"a", "b", "c", "new", "d", "e"},
		{"a", "b", "c", "d", "new", "e"},
		{"a", "b", "c", "d", "e", "new"},
	}
	for pick, w := range want {
		got := insertAt(order, checked, "new", func(n int) int {
			if n != 4 {
				t.Fatalf("n = %d, want 4 gaps", n)
			}
			return pick
		})
		if !slices.Equal(got, w) {
			t.Errorf("pick %d: %v, want %v", pick, got, w)
		}
	}
	if !slices.Equal(order, []string{"a", "b", "c", "d", "e"}) {
		t.Errorf("input mutated: %v", order)
	}
}

func TestInsertWhenEveryoneIsCheckedInAppends(t *testing.T) {
	t.Parallel()
	got := insertAt([]string{"a", "b"}, map[string]string{"a": "x", "b": "y"}, "new", func(int) int { return 0 })
	if !slices.Equal(got, []string{"a", "b", "new"}) {
		t.Errorf("got %v", got)
	}
}

func TestInsertIsIdempotent(t *testing.T) {
	t.Parallel()
	got := insertAt([]string{"a", "new"}, nil, "new", func(int) int { return 0 })
	if !slices.Equal(got, []string{"a", "new"}) {
		t.Errorf("got %v", got)
	}
}
