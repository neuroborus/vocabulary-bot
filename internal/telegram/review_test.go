package telegram

import (
	"strings"
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestFormatReviewReminderShowsSourceLabel(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord: "grasp",
		Anchors: []vocabulary.SourceAnchor{
			{
				Source:      vocabulary.SourcePocketBook,
				SourceLabel: "Necromancer — Fred Saberhagen",
			},
		},
		Contexts: []string{"Though it stood clear and sharp on the screen before him, Paul could not seem to grasp its image as a whole."},
	}, false)

	if !strings.Contains(text, "Necromancer — Fred Saberhagen") {
		t.Fatalf("source label missing: %q", text)
	}
	if strings.Index(text, "Necromancer") < strings.Index(text, "<b>Context</b>") {
		t.Fatalf("source label should appear after context: %q", text)
	}
	if !strings.HasSuffix(strings.TrimSpace(text), "Necromancer — Fred Saberhagen</i>") {
		t.Fatalf("source label should be at the bottom: %q", text)
	}
}

func TestFormatReviewReminderShowsSpreadsheetSourceLabel(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord:  "assessing",
		Translations: []string{"оценивание", "создание"},
		Contexts:     []string{"Assessing the risks took longer than we expected"},
		Anchors: []vocabulary.SourceAnchor{{
			Source:    vocabulary.SourceGoogleSheet,
			SheetName: "Vocabulary",
			RowNumber: 2,
		}},
	}, false)

	if !strings.Contains(text, "Vocabulary") {
		t.Fatalf("spreadsheet source label missing: %q", text)
	}
	if !strings.HasSuffix(strings.TrimSpace(text), "Vocabulary</i>") {
		t.Fatalf("spreadsheet source label should be at the bottom: %q", text)
	}
}

func TestFormatReviewReminderWrapsTranslationsInSpoiler(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord: "carve",
		Translations: []string{
			"[kɑ:v] Verb 1) вырезать",
			"2) резать",
		},
		Contexts: []string{"He began to carve the wood."},
	}, true)

	if !strings.Contains(text, "<tg-spoiler>") || !strings.Contains(text, "</tg-spoiler>") {
		t.Fatalf("translation spoiler missing: %q", text)
	}
	if strings.Index(text, "<b>Context</b>") > strings.Index(text, "<tg-spoiler>") {
		t.Fatalf("context should appear before translation spoiler: %q", text)
	}
	if strings.Index(text, "<tg-spoiler>") > strings.Index(text, "<b>Verb</b>") {
		t.Fatalf("translation section should be inside spoiler: %q", text)
	}
	if strings.Index(text, "</tg-spoiler>") < strings.Index(text, "вырезать") {
		t.Fatalf("translation meanings should be inside spoiler: %q", text)
	}
}

func TestFormatReviewReminderShowsTranslationsWhenSpoilerDisabled(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord: "carve",
		Translations: []string{
			"[kɑ:v] Verb 1) вырезать",
		},
		Contexts: []string{"He began to carve the wood."},
	}, false)

	if strings.Contains(text, "<tg-spoiler>") {
		t.Fatalf("spoiler should be disabled: %q", text)
	}
	if !strings.Contains(text, "<b>Verb</b>") || !strings.Contains(text, "вырезать") {
		t.Fatalf("translations should remain visible: %q", text)
	}
}

func TestFormatReviewReminderDictionaryEntry(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord: "lean",
		Forms:       []vocabulary.Form{{Value: "lean"}},
		Translations: []string{
			"[li:n] Noun 1) постное мясо",
			"2) низкооплачиваемая работа",
			"3) наклон",
			"4) что-л. невыгодное",
			"Verb 1) нагибаться",
			"2) сгибаться",
			"3) наклонять",
			"4) опираться",
			"5) полагаться",
			"to lean on a friend's advice - полагаться на совет друга",
			"Adjective 1) худой",
			"2) тощий",
			"3) постный",
			"о мясе",
			"4) скудный",
			"5) бедный",
			"о руднике",
		},
	}, true)

	for _, want := range []string{
		"<b>lean</b>",
		"<i>[li:n]</i>",
		"<b>Noun</b>",
		"постное мясо",
		"<b>Verb</b>",
		"нагибаться",
		"<b>Adjective</b>",
		"постный (о мясе)",
		"<b>Context</b>",
		"to lean on a friend's advice — полагаться на совет друга",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("reminder missing %q:\n%s", want, text)
		}
	}

	for _, unwanted := range []string{
		"<b>Forms</b>",
		"<b>Variants</b>",
		"<b>Translations</b>",
		"<b>Contexts</b>",
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("reminder should not include %q:\n%s", unwanted, text)
		}
	}
}

func TestFormatReviewReminderShowsVariantsOnlyWhenDifferent(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord: "to decelerate",
		Forms: []vocabulary.Form{
			{Value: "decelerate"},
			{Value: "to decelerate"},
		},
		Translations: []string{"замедляться"},
		Contexts:     []string{"The car began to decelerate."},
	}, true)

	if !strings.Contains(text, "<b>Variants</b>") || !strings.Contains(text, "decelerate") {
		t.Fatalf("variants missing: %q", text)
	}
	if !strings.Contains(text, "<b>Context</b>") || !strings.Contains(text, "The car began to decelerate.") {
		t.Fatalf("context section missing: %q", text)
	}
	if strings.Contains(text, "<b>Translation</b>\n• The car") {
		t.Fatalf("context should not be duplicated in translation section: %q", text)
	}
}

func TestFormatReviewReminderSimpleTranslation(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord:  "hello",
		Translations: []string{"привет", "здравствуйте"},
	}, true)

	if !strings.Contains(text, "<b>Translation</b>") {
		t.Fatalf("simple translation section missing: %q", text)
	}
	if strings.Contains(text, "<b>Noun</b>") {
		t.Fatalf("unexpected POS section: %q", text)
	}
}

func TestFormatReviewReminderAttachesRussianQualifiers(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord: "carve",
		Translations: []string{
			"[kɑ:v] Verb 1) вырезать",
			"2) резать",
			"по дереву или кости",
			"3) высекать",
			"из камня",
			"4) гравировать",
			"5) разрезать",
		},
	}, true)

	for _, want := range []string{
		"<b>carve</b>",
		"<i>[kɑ:v]</i>",
		"<b>Verb</b>",
		"вырезать",
		"резать (по дереву или кости)",
		"высекать (из камня)",
		"гравировать",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("reminder missing %q:\n%s", want, text)
		}
	}

	for _, unwanted := range []string{
		"• по дереву или кости",
		"• из камня",
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("reminder should not list qualifier as separate bullet %q:\n%s", unwanted, text)
		}
	}
}

func TestFormatReviewReminderCleansDanglingBookQuotes(t *testing.T) {
	t.Parallel()

	text := formatReviewReminder(vocabulary.Item{
		DisplayWord: "sardonic",
		Translations: []string{
			"[sɑ:'dɔnɪk] Adjective 1) злобный",
			"2) сардонический",
			"3) язвительный",
		},
		Contexts: []string{
			`" The voice from the lips was deep and pleasantly sardonic.`,
		},
	}, true)

	if strings.Contains(text, `" The voice`) {
		t.Fatalf("reminder should not keep dangling opening quote: %q", text)
	}
	if !strings.Contains(text, "The voice from the lips was deep and pleasantly sardonic.") {
		t.Fatalf("cleaned book context missing: %q", text)
	}
}

func TestParseLexiconDisplayExtractsTranscriptionAndPOS(t *testing.T) {
	t.Parallel()

	display := parseLexiconDisplay([]string{
		"[li:n] Verb 1) наклонять",
		"Adjective 1) худой",
	}, nil)

	if display.Transcription != "li:n" {
		t.Fatalf("transcription = %q, want li:n", display.Transcription)
	}
	if len(display.PartsOfSpeech) != 2 {
		t.Fatalf("parts of speech = %d, want 2", len(display.PartsOfSpeech))
	}
	if display.PartsOfSpeech[0].Label != "Verb" {
		t.Fatalf("first POS = %q, want Verb", display.PartsOfSpeech[0].Label)
	}
}

func TestParseReviewCallback(t *testing.T) {
	t.Parallel()

	normalizedKey := "decelerate"
	action, token, ok := parseReviewCallback(reviewCallbackData(reviewActionHard, normalizedKey))
	if !ok {
		t.Fatal("parseReviewCallback() = false, want true")
	}
	if action != reviewActionHard {
		t.Fatalf("action = %q, want %q", action, reviewActionHard)
	}
	if token != reviewCallbackToken(normalizedKey) {
		t.Fatalf("token = %q, want %q", token, reviewCallbackToken(normalizedKey))
	}
	if strings.Contains(token, normalizedKey) {
		t.Fatalf("token must not embed normalized key: %q", token)
	}
}

func TestReviewKeyboardHasEasyAndHardOnly(t *testing.T) {
	t.Parallel()

	keyboard := reviewKeyboard("decelerate")
	if len(keyboard.InlineKeyboard) != 1 || len(keyboard.InlineKeyboard[0]) != 2 {
		t.Fatalf("keyboard = %#v, want one row with Easy and Hard", keyboard.InlineKeyboard)
	}
}

func TestFormatReviewAnsweredAppendsChoice(t *testing.T) {
	t.Parallel()

	text := formatReviewAnswered(vocabulary.Item{
		DisplayWord: "decelerate",
		Review: vocabulary.ReviewState{
			IntervalDays: 2,
		},
	}, reviewActionEasy, false)

	if !strings.Contains(text, "decelerate") {
		t.Fatalf("text = %q", text)
	}
	if !strings.Contains(text, "✓ Easy") {
		t.Fatalf("text = %q, want Easy marker", text)
	}
	if !strings.Contains(text, "2 day(s)") {
		t.Fatalf("text = %q, want interval", text)
	}
}

func TestReviewKeyboardCallbackDataWithinTelegramLimit(t *testing.T) {
	t.Parallel()

	longKey := strings.Repeat("assessingtherisksandbenefitsoftheproposedtrial", 4)
	keyboard := reviewKeyboard(longKey)
	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			if len(button.CallbackData) > 64 {
				t.Fatalf("callback data too long (%d): %q", len(button.CallbackData), button.CallbackData)
			}
			if strings.Contains(button.CallbackData, longKey) {
				t.Fatalf("callback data must not embed normalized key: %q", button.CallbackData)
			}
		}
	}
}
