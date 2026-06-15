package pocketbook

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAnchorPositionDictionaryWord(t *testing.T) {
	t.Parallel()

	info, ok := ParseAnchorPosition("pbr:/word?page=11&offs=518")
	if !ok {
		t.Fatal("ParseAnchorPosition() = false")
	}
	if info.Kind != "word" {
		t.Fatalf("Kind = %q, want word", info.Kind)
	}
	if info.Page != "11" {
		t.Fatalf("Page = %q, want 11", info.Page)
	}
	if info.Offset != 518 {
		t.Fatalf("Offset = %d, want 518", info.Offset)
	}
}

func TestIsDictionaryWordAnchor(t *testing.T) {
	t.Parallel()

	if !IsDictionaryWordAnchor("pbr:/word?page=11&offs=518") {
		t.Fatal("expected dictionary word anchor")
	}
	if IsDictionaryWordAnchor("pbr:/page?page=36&offs=123") {
		t.Fatal("page anchor should not match dictionary word anchor")
	}
}

func TestFlattenEPUBAndExtractSentenceAtOffset(t *testing.T) {
	t.Parallel()

	body := `<html><body><p>He had to lean against the wall. Then he left.</p></body></html>`
	epubPath := writeTestEPUB(t, body)

	text, err := FlattenEPUB(epubPath)
	if err != nil {
		t.Fatalf("FlattenEPUB() error = %v", err)
	}

	offset := bytes.Index([]byte(text), []byte("lean"))
	if offset < 0 {
		t.Fatalf("flattened text = %q, expected lean", text)
	}

	sentence, ok := ExtractSentenceAtOffset(text, offset)
	if !ok {
		t.Fatalf("ExtractSentenceAtOffset() = false for text %q", text)
	}
	if sentence != "He had to lean against the wall." {
		t.Fatalf("sentence = %q", sentence)
	}
}

func TestExtractSentenceAtOffsetStripsDanglingDialogueQuote(t *testing.T) {
	t.Parallel()

	text := `Earlier sentence. " The voice from the lips was deep and pleasantly sardonic. Another one.`
	offset := strings.Index(text, "sardonic")
	if offset < 0 {
		t.Fatal("test text missing target word")
	}

	sentence, ok := ExtractSentenceAtOffset(text, offset)
	if !ok {
		t.Fatal("ExtractSentenceAtOffset() = false")
	}
	if sentence != "The voice from the lips was deep and pleasantly sardonic." {
		t.Fatalf("sentence = %q", sentence)
	}
}

func writeTestEPUB(t *testing.T, bodyHTML string) string {
	t.Helper()

	epubPath := filepath.Join(t.TempDir(), "sample.epub")
	buffer := new(bytes.Buffer)
	writer := zip.NewWriter(buffer)

	writeZipEntry(t, writer, "mimetype", "application/epub+zip")
	writeZipEntry(t, writer, "META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	writeZipEntry(t, writer, "OEBPS/content.opf", `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`)
	writeZipEntry(t, writer, "OEBPS/chapter1.xhtml", bodyHTML)

	if err := writer.Close(); err != nil {
		t.Fatalf("close test epub zip: %v", err)
	}
	if err := os.WriteFile(epubPath, buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("write test epub: %v", err)
	}

	return epubPath
}

func TestNormalizeBookTextCollapsesWhitespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "duplicate newlines",
			input: "a\n\n\nb",
			want:  "a\nb",
		},
		{
			name:  "leading and trailing whitespace",
			input: "  hello   world \n\n",
			want:  "hello world",
		},
		{
			name:  "hyphenated line break",
			input: "They used a tea-\nsel to raise the nap.",
			want:  "They used a teasel to raise the nap.",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeBookText(test.input); got != test.want {
				t.Fatalf("normalizeBookText() = %q, want %q", got, test.want)
			}
		})
	}
}

func writeZipEntry(t *testing.T, writer *zip.Writer, name, content string) {
	t.Helper()

	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %q: %v", name, err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		t.Fatalf("write zip entry %q: %v", name, err)
	}
}
