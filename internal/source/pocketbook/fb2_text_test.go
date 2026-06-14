package pocketbook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlattenFB2ExtractsParagraphText(t *testing.T) {
	t.Parallel()

	fb2Path := filepath.Join(t.TempDir(), "sample.fb2")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <body>
    <section>
      <p>He was frowning at the newcomer.</p>
      <p>Another paragraph followed.</p>
    </section>
  </body>
</FictionBook>`
	if err := os.WriteFile(fb2Path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fb2: %v", err)
	}

	text, err := FlattenFB2(fb2Path)
	if err != nil {
		t.Fatalf("FlattenFB2() error = %v", err)
	}

	offset := findSubstringOffset(text, "frowning")
	sentence, ok := ExtractSentenceAtOffset(text, offset)
	if !ok {
		t.Fatalf("ExtractSentenceAtOffset() = false for %q", text)
	}
	if sentence != "He was frowning at the newcomer." {
		t.Fatalf("sentence = %q", sentence)
	}
}

func TestBookTextFormatDetectsFB2(t *testing.T) {
	t.Parallel()

	book := Book{
		Path:     "/books/Necromancer.fb2",
		MimeType: "application/x-fictionbook+xml",
	}
	if got := bookTextFormat(book); got != "fb2" {
		t.Fatalf("bookTextFormat() = %q, want fb2", got)
	}
	if !supportsBookContext(book) {
		t.Fatal("supportsBookContext() = false, want true")
	}
}
