package spreadsheet

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func ParseServiceAccountCredentials(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("service account credentials are empty")
	}

	if json.Valid([]byte(raw)) {
		return []byte(raw), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode service account credentials: %w", err)
	}
	if !json.Valid(decoded) {
		return nil, fmt.Errorf("service account credentials are not valid JSON")
	}

	return decoded, nil
}
