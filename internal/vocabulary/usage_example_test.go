package vocabulary

import "testing"

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
