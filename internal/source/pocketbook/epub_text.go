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
	"unicode/utf8"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

var blockBoundaryPattern = regexp.MustCompile(`(?i)</(p|div|li|h[1-6]|blockquote|section|article)>`)
var hyphenatedLineBreakPattern = regexp.MustCompile(`([[:alpha:]])-\n([[:alpha:]])`)

const wordOffsetSearchWindow = 512

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
	value = strings.ReplaceAll(value, "\u00ad", "")
	value = hyphenatedLineBreakPattern.ReplaceAllString(value, "$1$2")

	var builder strings.Builder
	builder.Grow(len(value))
	previousSpace := false
	previousNewline := true

	for _, r := range value {
		switch {
		case r == '\n':
			if previousNewline {
				continue
			}
			builder.WriteByte('\n')
			previousSpace = false
			previousNewline = true
		case unicode.IsSpace(r):
			if builder.Len() == 0 || previousSpace {
				continue
			}
			builder.WriteByte(' ')
			previousSpace = true
			previousNewline = false
		default:
			builder.WriteRune(r)
			previousSpace = false
			previousNewline = false
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
	if isDictionaryWordMatch(text, offset, word) {
		return true
	}

	// PocketBook offsets may point at whitespace before the selected token.
	trimmed := strings.TrimLeft(candidate, " \t\n\r")
	if trimmed == candidate {
		return false
	}

	trimOffset := offset + (len(candidate) - len(trimmed))
	return isDictionaryWordMatch(text, trimOffset, word)
}

func resolveWordOffset(text, word string, preferredOffset int) (int, bool) {
	if TextMatchesWordAtOffset(text, preferredOffset, word) {
		return preferredOffset, true
	}

	if offset, ok := findWordNearOffset(text, word, preferredOffset); ok {
		return offset, true
	}

	positions := allWordPositions(text, word)
	if len(positions) == 0 {
		if sentence, ok := ExtractSentenceAtOffset(text, preferredOffset); ok && sentenceContainsWord(sentence, word) {
			return preferredOffset, true
		}
		return 0, false
	}

	return nearestWordPosition(preferredOffset, positions), true
}

func allWordPositions(text, word string) []int {
	positions := findWordPositions(text, word)
	if len(positions) > 0 {
		return positions
	}

	return findWordPositionsFlexible(text, word)
}

func nearestWordPosition(preferredOffset int, positions []int) int {
	best := positions[0]
	bestDistance := absInt(positions[0] - preferredOffset)
	for _, position := range positions[1:] {
		distance := absInt(position - preferredOffset)
		if distance < bestDistance {
			best = position
			bestDistance = distance
		}
	}

	return best
}

func countWordOccurrences(text, word string) int {
	return len(allWordPositions(text, word))
}

func sentenceContainsWord(sentence, word string) bool {
	word = strings.TrimSpace(word)
	if word == "" {
		return false
	}

	return len(findWordPositions(sentence, word)) > 0 ||
		len(findWordPositionsFlexible(sentence, word)) > 0
}

func findWordPositionsFlexible(text, word string) []int {
	collapsed, indexMap := collapseHyphenation(text)
	if len(indexMap) == 0 {
		return nil
	}

	positions := findWordPositions(collapsed, word)
	if len(positions) == 0 {
		return nil
	}

	mapped := make([]int, 0, len(positions))
	for _, position := range positions {
		if position < 0 || position >= len(indexMap) {
			continue
		}
		mapped = append(mapped, indexMap[position])
	}

	return mapped
}

func collapseHyphenation(text string) (string, []int) {
	text = strings.ReplaceAll(text, "\u00ad", "")

	var (
		builder  strings.Builder
		indexMap []int
	)
	builder.Grow(len(text))
	indexMap = make([]int, 0, len(text))

	for byteIndex := 0; byteIndex < len(text); {
		if text[byteIndex] == '-' && byteIndex+1 < len(text) && text[byteIndex+1] == '\n' {
			byteIndex++
			continue
		}
		if text[byteIndex] == '\n' || text[byteIndex] == '\r' {
			byteIndex++
			continue
		}

		r, size := utf8.DecodeRuneInString(text[byteIndex:])
		indexMap = append(indexMap, byteIndex)
		builder.WriteRune(r)
		byteIndex += size
	}

	return builder.String(), indexMap
}

func findWordNearOffset(text, word string, preferredOffset int) (int, bool) {
	start := preferredOffset - wordOffsetSearchWindow
	if start < 0 {
		start = 0
	}
	end := preferredOffset + wordOffsetSearchWindow
	if end > len(text) {
		end = len(text)
	}

	segment := text[start:end]
	positions := allWordPositions(segment, word)
	if len(positions) == 0 {
		return 0, false
	}

	absolute := make([]int, len(positions))
	for index, position := range positions {
		absolute[index] = start + position
	}

	return nearestWordPosition(preferredOffset, absolute), true
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
		if isDictionaryWordMatch(text, absolute, word) {
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
		if text[end] != '\'' || !hasPossessiveSuffix(text, start, length) {
			return false
		}
	}

	return true
}

func isDictionaryWordMatch(text string, start int, word string) bool {
	word = strings.TrimSpace(word)
	if word == "" || start < 0 || start >= len(text) {
		return false
	}

	candidate := text[start:]
	if !strings.HasPrefix(strings.ToLower(candidate), strings.ToLower(word)) {
		return false
	}
	if start > 0 && isWordCharacter(text[start-1]) {
		return false
	}

	rest := candidate[len(word):]
	if rest == "" {
		return true
	}

	for index := 0; index < len(rest); {
		r, size := utf8.DecodeRuneInString(rest[index:])
		if unicode.IsLetter(r) || r == '\'' {
			index += size
			continue
		}

		return true
	}

	return true
}

func hasPossessiveSuffix(text string, start, length int) bool {
	end := start + length
	if end >= len(text) || text[end] != '\'' {
		return false
	}
	if end+1 >= len(text) {
		return true
	}
	if text[end+1] != 's' && text[end+1] != 'S' {
		return false
	}

	after := end + 2
	if after >= len(text) {
		return true
	}

	return !isWordCharacter(text[after])
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
