package pocketbook

import (
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
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open fb2: %w", err)
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
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
				inBody = true
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
