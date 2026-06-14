package vocabulary

import (
	"reflect"
	"testing"
)

func TestBuildLookupKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		word string
		want []string
	}{
		{
			name: "single token",
			word: "decelerate",
			want: []string{"decelerate"},
		},
		{
			name: "leading edge token",
			word: "to decelerate",
			want: []string{"todecelerate", "decelerate"},
		},
		{
			name: "trailing edge token",
			word: "decelerate to",
			want: []string{"decelerateto", "decelerate"},
		},
		{
			name: "leading and trailing edge tokens",
			word: "the same thing",
			want: []string{"thesamething", "samething", "thesame", "same"},
		},
		{
			name: "space and dash compact to same key",
			word: "Ice\u2011cream",
			want: []string{"icecream"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := BuildLookupKeys(test.word)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("BuildLookupKeys(%q) = %#v, want %#v", test.word, got, test.want)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	t.Parallel()

	got := NormalizeText("  (ICE\u2011cream)  ")
	want := "ice-cream"

	if got != want {
		t.Fatalf("NormalizeText() = %q, want %q", got, want)
	}
}
