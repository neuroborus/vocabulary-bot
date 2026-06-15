package pocketbook

import (
	"archive/zip"
	"bytes"
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

func TestFlattenFB2FromZipArchive(t *testing.T) {
	t.Parallel()

	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <body>
    <section>
      <p>He used a teasel on the cloth.</p>
    </section>
  </body>
</FictionBook>`

	zipPath := filepath.Join(t.TempDir(), "sample.fb2.zip")
	buffer := new(bytes.Buffer)
	writer := zip.NewWriter(buffer)
	entry, err := writer.Create("Necromancer.fb2")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := entry.Write([]byte(fb2Content)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := os.WriteFile(zipPath, buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("write zip file: %v", err)
	}

	text, err := FlattenFB2(zipPath)
	if err != nil {
		t.Fatalf("FlattenFB2() error = %v", err)
	}

	offset := findSubstringOffset(text, "teasel")
	if offset < 0 {
		t.Fatalf("flattened text = %q, expected teasel", text)
	}

	sentence, ok := ExtractSentenceAtOffset(text, offset)
	if !ok {
		t.Fatalf("ExtractSentenceAtOffset() = false for %q", text)
	}
	if sentence != "He used a teasel on the cloth." {
		t.Fatalf("sentence = %q", sentence)
	}
}

func TestFlattenFB2SkipsNotesBody(t *testing.T) {
	t.Parallel()

	fb2Path := filepath.Join(t.TempDir(), "sample.fb2")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <body name="notes">
    <p>notes-only carve mention</p>
  </body>
  <body>
    <p>Main text with gauge reading.</p>
  </body>
</FictionBook>`
	if err := os.WriteFile(fb2Path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fb2: %v", err)
	}

	text, err := FlattenFB2(fb2Path)
	if err != nil {
		t.Fatalf("FlattenFB2() error = %v", err)
	}
	if stringsContains(text, "notes-only carve") {
		t.Fatalf("notes body leaked into flattened text: %q", text)
	}
	if !stringsContains(text, "gauge") {
		t.Fatalf("main body missing from flattened text: %q", text)
	}
}

func stringsContains(text, needle string) bool {
	return findSubstringOffset(text, needle) >= 0
}
