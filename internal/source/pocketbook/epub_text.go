package pocketbook

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

var blockBoundaryPattern = regexp.MustCompile(`(?i)</(p|div|li|h[1-6]|blockquote|section|article)>`)

func FlattenEPUB(epubPath string) (string, error) {
	reader, err := zip.OpenReader(epubPath)
	if err != nil {
		return "", fmt.Errorf("open epub zip: %w", err)
	}
	defer reader.Close()

	rootPath, err := epubRootPath(reader.File)
	if err != nil {
		return "", err
	}

	spineIDs, err := epubSpineItemIDs(reader.File, rootPath)
	if err != nil {
		return "", err
	}

	manifest, err := epubManifest(reader.File, rootPath)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, id := range spineIDs {
		href, ok := manifest[id]
		if !ok {
			continue
		}

		contentPath := path.Join(path.Dir(rootPath), href)
		data, err := readZipFile(reader.File, contentPath)
		if err != nil {
			continue
		}

		plain := htmlToPlainText(string(data))
		if plain == "" {
			continue
		}

		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(plain)
	}

	text := normalizeBookText(builder.String())
	if text == "" {
		return "", fmt.Errorf("epub contains no readable text")
	}

	return text, nil
}

func ExtractSentenceAtOffset(text string, offset int) (string, bool) {
	if offset < 0 || offset >= len(text) {
		return "", false
	}

	start := offset
	for start > 0 {
		previous := text[start-1]
		if previous == '\n' || isSentenceBoundary(previous) {
			break
		}
		start--
	}

	end := offset
	for end < len(text) {
		current := text[end]
		if current == '\n' {
			break
		}
		if isSentenceBoundary(current) {
			end++
			break
		}
		end++
	}

	sentence := vocabulary.CleanContextLine(text[start:end])
	if !looksLikeContext(sentence) {
		return "", false
	}

	return sentence, true
}

func htmlToPlainText(value string) string {
	value = htmlTagPattern.ReplaceAllString(value, " ")
	value = blockBoundaryPattern.ReplaceAllString(value, "\n")
	value = strings.ReplaceAll(value, "&nbsp;", " ")
	value = strings.ReplaceAll(value, "&amp;", "&")
	value = strings.ReplaceAll(value, "&lt;", "<")
	value = strings.ReplaceAll(value, "&gt;", ">")
	value = strings.ReplaceAll(value, "&quot;", "\"")
	value = strings.ReplaceAll(value, "&#39;", "'")

	return normalizeBookText(value)
}

func normalizeBookText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	var builder strings.Builder
	builder.Grow(len(value))
	previousSpace := false

	for _, r := range value {
		switch {
		case r == '\n':
			if builder.Len() == 0 || strings.HasSuffix(builder.String(), "\n") {
				continue
			}
			builder.WriteByte('\n')
			previousSpace = false
		case unicode.IsSpace(r):
			if builder.Len() == 0 || previousSpace {
				continue
			}
			builder.WriteByte(' ')
			previousSpace = true
		default:
			builder.WriteRune(r)
			previousSpace = false
		}
	}

	return strings.TrimSpace(builder.String())
}

func isSentenceBoundary(value byte) bool {
	return value == '.' || value == '!' || value == '?'
}

func epubRootPath(files []*zip.File) (string, error) {
	data, err := readZipFile(files, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("read container.xml: %w", err)
	}

	type rootFile struct {
		FullPath string `xml:"full-path,attr"`
	}
	type rootfiles struct {
		RootFiles []rootFile `xml:"rootfiles>rootfile"`
	}

	var container rootfiles
	if err := xml.Unmarshal(data, &container); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}
	if len(container.RootFiles) == 0 || container.RootFiles[0].FullPath == "" {
		return "", fmt.Errorf("container.xml has no rootfile")
	}

	return container.RootFiles[0].FullPath, nil
}

func epubManifest(files []*zip.File, rootPath string) (map[string]string, error) {
	data, err := readZipFile(files, rootPath)
	if err != nil {
		return nil, fmt.Errorf("read opf: %w", err)
	}

	type manifestItem struct {
		ID   string `xml:"id,attr"`
		Href string `xml:"href,attr"`
	}
	type packageDocument struct {
		Manifest struct {
			Items []manifestItem `xml:"item"`
		} `xml:"manifest"`
	}

	var document packageDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse opf manifest: %w", err)
	}

	manifest := make(map[string]string, len(document.Manifest.Items))
	for _, item := range document.Manifest.Items {
		if item.ID == "" || item.Href == "" {
			continue
		}
		manifest[item.ID] = item.Href
	}

	if len(manifest) == 0 {
		return nil, fmt.Errorf("opf manifest is empty")
	}

	return manifest, nil
}

func epubSpineItemIDs(files []*zip.File, rootPath string) ([]string, error) {
	data, err := readZipFile(files, rootPath)
	if err != nil {
		return nil, fmt.Errorf("read opf: %w", err)
	}

	type spineItem struct {
		IDRef string `xml:"idref,attr"`
	}
	type packageDocument struct {
		Spine struct {
			Items []spineItem `xml:"itemref"`
		} `xml:"spine"`
	}

	var document packageDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse opf spine: %w", err)
	}

	ids := make([]string, 0, len(document.Spine.Items))
	for _, item := range document.Spine.Items {
		if item.IDRef == "" {
			continue
		}
		ids = append(ids, item.IDRef)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("opf spine is empty")
	}

	return ids, nil
}

func readZipFile(files []*zip.File, name string) ([]byte, error) {
	normalized := path.Clean(name)
	for _, file := range files {
		if path.Clean(file.Name) != normalized {
			continue
		}

		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		return data, nil
	}

	return nil, fmt.Errorf("zip entry %q not found", name)
}

func TextMatchesWordAtOffset(text string, offset int, word string) bool {
	word = strings.TrimSpace(word)
	if word == "" || offset < 0 || offset >= len(text) {
		return false
	}

	candidate := text[offset:]
	if strings.HasPrefix(candidate, word) {
		return true
	}

	// PocketBook offsets may point at whitespace before the selected token.
	trimmed := strings.TrimLeft(candidate, " \t\n\r")
	return strings.HasPrefix(trimmed, word)
}

func resolveWordOffset(text, word string, preferredOffset int) (int, bool) {
	if TextMatchesWordAtOffset(text, preferredOffset, word) {
		return preferredOffset, true
	}

	positions := findWordPositions(text, word)
	if len(positions) == 0 {
		return 0, false
	}
	if len(positions) == 1 {
		return positions[0], true
	}

	best := positions[0]
	bestDistance := absInt(positions[0] - preferredOffset)
	for _, position := range positions[1:] {
		distance := absInt(position - preferredOffset)
		if distance < bestDistance {
			best = position
			bestDistance = distance
		}
	}

	return best, true
}

func findWordPositions(text, word string) []int {
	word = strings.TrimSpace(word)
	if word == "" {
		return nil
	}

	lowerText := strings.ToLower(text)
	lowerWord := strings.ToLower(word)

	positions := make([]int, 0, 4)
	searchFrom := 0
	for {
		index := strings.Index(lowerText[searchFrom:], lowerWord)
		if index < 0 {
			break
		}

		absolute := searchFrom + index
		if isWordBoundaryMatch(text, absolute, len(word)) {
			positions = append(positions, absolute)
		}

		searchFrom = absolute + len(lowerWord)
	}

	return positions
}

func isWordBoundaryMatch(text string, start int, length int) bool {
	if start > 0 && isWordCharacter(text[start-1]) {
		return false
	}

	end := start + length
	if end < len(text) && isWordCharacter(text[end]) {
		return false
	}

	return true
}

func isWordCharacter(value byte) bool {
	return (value >= 'a' && value <= 'z') ||
		(value >= 'A' && value <= 'Z') ||
		(value >= '0' && value <= '9') ||
		value == '\''
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
