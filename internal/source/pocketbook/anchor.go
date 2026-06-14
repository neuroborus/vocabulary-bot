package pocketbook

import (
	"net/url"
	"strconv"
	"strings"
)

type AnchorPosition struct {
	Kind   string
	Page   string
	Offset int
}

func ParseAnchorPosition(position string) (AnchorPosition, bool) {
	position = strings.TrimSpace(position)
	if position == "" {
		return AnchorPosition{}, false
	}

	kind, query := splitPBRAnchor(position)
	if kind == "" {
		return AnchorPosition{}, false
	}

	info := AnchorPosition{Kind: kind}
	values, err := url.ParseQuery(query)
	if err != nil {
		values = manualAnchorQuery(query)
	}

	if page := values.Get("page"); page != "" {
		info.Page = page
	}

	offsetValue := values.Get("offs")
	if offsetValue == "" {
		return info, true
	}

	offset, err := strconv.Atoi(offsetValue)
	if err != nil || offset < 0 {
		return info, true
	}

	info.Offset = offset
	return info, true
}

func splitPBRAnchor(position string) (kind string, query string) {
	cleaned := strings.TrimPrefix(position, "pbr:/")
	if cleaned == position {
		cleaned = strings.TrimPrefix(position, "pbr:")
		cleaned = strings.TrimLeft(cleaned, "/")
	}

	parts := strings.SplitN(cleaned, "?", 2)
	kind = strings.Trim(parts[0], "/")
	if len(parts) == 2 {
		query = parts[1]
	}

	return kind, query
}

func manualAnchorQuery(query string) url.Values {
	values := make(url.Values)
	for _, part := range strings.Split(query, "&") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		values.Set(key, value)
	}

	return values
}

func IsDictionaryWordAnchor(position string) bool {
	info, ok := ParseAnchorPosition(position)
	return ok && info.Kind == "word" && info.Offset > 0
}
