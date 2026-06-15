package pocketbook

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

var fb2TextElements = map[string]struct{}{
	"p":        {},
	"v":        {},
	"subtitle": {},
	"text":     {},
	"emphasis": {},
	"strong":   {},
}

func FlattenFB2(path string) (string, error) {
	reader, cleanup, err := openFB2Reader(path)
	if err != nil {
		return "", err
	}
	defer cleanup()

	return flattenFB2XML(reader)
}

func openFB2Reader(path string) (io.ReadCloser, func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open fb2: %w", err)
	}

	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		_ = file.Close()
		return nil, func() {}, fmt.Errorf("read fb2 header: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, func() {}, fmt.Errorf("rewind fb2: %w", err)
	}

	if !isZipHeader(header) {
		return file, func() { _ = file.Close() }, nil
	}
	_ = file.Close()

	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open fb2 zip: %w", err)
	}

	entry, err := findFB2ZipEntry(archive.File)
	if err != nil {
		_ = archive.Close()
		return nil, func() {}, err
	}

	reader, err := entry.Open()
	if err != nil {
		_ = archive.Close()
		return nil, func() {}, fmt.Errorf("open fb2 zip entry %q: %w", entry.Name, err)
	}

	return reader, func() {
		_ = reader.Close()
		_ = archive.Close()
	}, nil
}

func isZipHeader(header []byte) bool {
	return len(header) >= 2 && header[0] == 'P' && header[1] == 'K'
}

func findFB2ZipEntry(files []*zip.File) (*zip.File, error) {
	var fb2Entries []*zip.File
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(file.Name)), ".fb2") {
			fb2Entries = append(fb2Entries, file)
		}
	}
	if len(fb2Entries) == 1 {
		return fb2Entries[0], nil
	}
	if len(fb2Entries) > 1 {
		return fb2Entries[0], nil
	}

	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			continue
		}
		prefix := make([]byte, 5)
		_, _ = io.ReadFull(reader, prefix)
		_ = reader.Close()
		if strings.HasPrefix(string(prefix), "<?xml") || strings.HasPrefix(string(prefix), "<Fict") {
			return file, nil
		}
	}

	return nil, fmt.Errorf("fb2 zip contains no readable fictionbook entry")
}

func flattenFB2XML(reader io.Reader) (string, error) {
	decoder := xml.NewDecoder(reader)
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	var (
		inBody       bool
		captureDepth int
		builder      strings.Builder
		paragraph    strings.Builder
	)

	flushParagraph := func() {
		paragraphText := normalizeBookText(paragraph.String())
		paragraph.Reset()
		if paragraphText == "" {
			return
		}
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(paragraphText)
	}

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("decode fb2 xml: %w", err)
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local == "body" {
				inBody = !isSecondaryFB2Body(bodyNameAttr(element))
			}
			if inBody {
				if _, ok := fb2TextElements[element.Name.Local]; ok {
					captureDepth++
				}
			}
		case xml.CharData:
			if inBody && captureDepth > 0 {
				paragraph.Write(element)
			}
		case xml.EndElement:
			if element.Name.Local == "body" {
				flushParagraph()
				inBody = false
			}
			if inBody {
				if _, ok := fb2TextElements[element.Name.Local]; ok {
					if captureDepth > 0 {
						captureDepth--
					}
					if captureDepth == 0 {
						flushParagraph()
					}
				}
			}
		}
	}

	text := normalizeBookText(builder.String())
	if text == "" {
		return "", fmt.Errorf("fb2 contains no readable text")
	}

	return text, nil
}

func bodyNameAttr(element xml.StartElement) string {
	for _, attr := range element.Attr {
		if attr.Name.Local == "name" {
			return strings.ToLower(strings.TrimSpace(attr.Value))
		}
	}

	return ""
}

func isSecondaryFB2Body(name string) bool {
	switch name {
	case "notes", "comments":
		return true
	default:
		return false
	}
}
