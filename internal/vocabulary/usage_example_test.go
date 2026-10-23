package vocabulary

import (
	"reflect"
	"testing"
)

func TestPartitionUsageExamplesMovesDictionaryUsageLineToContexts(t *testing.T) {
	t.Parallel()

	translations, contexts := PartitionUsageExamples([]string{
		"Verb 1) наклонять",
		"to lean on a friend's advice - полагаться на совет друга",
		"Adjective 1) худой",
	})

	if len(translations) != 2 {
		t.Fatalf("translations = %#v, want 2 dictionary lines", translations)
	}
	if len(contexts) != 1 {
		t.Fatalf("contexts = %#v, want 1 usage example", contexts)
	}
	if contexts[0] != "to lean on a friend's advice — полагаться на совет друга" {
		t.Fatalf("context = %q", contexts[0])
	}
}

func TestNormalizeUsageExamplesBackfillsStoredItem(t *testing.T) {
	t.Parallel()

	item := Item{
		Translations: []string{
			"[li:n] Noun 1) постное мясо",
			"to lean on a friend's advice - полагаться на совет друга",
		},
	}

	NormalizeUsageExamples(&item)

	if len(item.Translations) != 1 {
		t.Fatalf("translations = %#v", item.Translations)
	}
	if len(item.Contexts) != 1 {
		t.Fatalf("contexts = %#v", item.Contexts)
	}
}

func TestIsUsageExampleLineRequiresCyrillicTranslation(t *testing.T) {
	t.Parallel()

	if !IsUsageExampleLine("to lean on a friend's advice - полагаться на совет друга") {
		t.Fatal("expected dictionary usage example with Russian translation")
	}
	if IsUsageExampleLine("But this man - this Guild-master - was nothing so simple as a crackpot.") {
		t.Fatal("English book sentence with dashes should not be a usage example")
	}
}

func TestUsageExampleSeparators(t *testing.T) {
	t.Parallel()

	for _, separator := range []string{" - ", " – ", " — "} {
		t.Run(separator, func(t *testing.T) {
			line := "to lean on a friend's advice" + separator + "полагаться на совет друга"
			if !IsUsageExampleLine(line) {
				t.Fatalf("usage example not recognized: %q", line)
			}
			want := "to lean on a friend's advice — полагаться на совет друга"
			if got := FormatUsageExampleLine(line); got != want {
				t.Fatalf("FormatUsageExampleLine() = %q, want %q", got, want)
			}
			translations, contexts := PartitionUsageExamples([]string{"полагаться", line, want})
			if !reflect.DeepEqual(translations, []string{"полагаться"}) || !reflect.DeepEqual(contexts, []string{want}) {
				t.Fatalf("partition = %#v, %#v; want translation and one normalized example", translations, contexts)
			}
			if HasBookLikeContext([]string{line}) {
				t.Fatal("bilingual example must not suppress book-context enrichment")
			}

			dialogue := "But this man" + separator + "this Guild-master" + separator + "was nothing so simple as a crackpot."
			if IsUsageExampleLine(dialogue) || FormatUsageExampleLine(dialogue) != dialogue || !HasBookLikeContext([]string{dialogue}) {
				t.Fatalf("ordinary English dash sentence misclassified or rewritten: %q", dialogue)
			}
			if !HasBookLikeContext([]string{line, dialogue}) {
				t.Fatal("a real book sentence alongside an example must still be detected")
			}
		})
	}
}

func TestSplitUsageExamplePreservesDashesWithinBothLanguages(t *testing.T) {
	t.Parallel()

	line := "He leaned on it — carefully — он опёрся — осторожно"
	example, translation, ok := SplitUsageExampleLine(line)
	if !ok || example != "He leaned on it — carefully" || translation != "он опёрся — осторожно" {
		t.Fatalf("SplitUsageExampleLine() = %q, %q, %v", example, translation, ok)
	}
}

func TestShortBilingualLinesSeparatePresentationFromStorage(t *testing.T) {
	t.Parallel()

	for _, separator := range []string{" - ", " – ", " — "} {
		for _, pair := range []struct {
			example     string
			translation string
		}{
			{"I agree", "Я согласен"},
			{"Hello", "Привет"},
		} {
			line := pair.example + separator + pair.translation
			example, translation, ok := SplitUsageExampleLine(line)
			if !ok || example != pair.example || translation != pair.translation {
				t.Fatalf("SplitUsageExampleLine(%q) = %q, %q, %v", line, example, translation, ok)
			}
			if IsUsageExampleLine(line) {
				t.Fatalf("short line must retain conservative storage classification: %q", line)
			}
			translations, contexts := PartitionUsageExamples([]string{line})
			if !reflect.DeepEqual(translations, []string{line}) || len(contexts) != 0 {
				t.Fatalf("storage partition changed for %q: %#v, %#v", line, translations, contexts)
			}
		}
	}
	if _, _, ok := SplitUsageExampleLine("I agree — you win"); ok {
		t.Fatal("ordinary English dash dialogue must not be split as bilingual")
	}
}
