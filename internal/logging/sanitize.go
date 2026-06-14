package logging

import (
	"regexp"
)

var sanitizePatterns = []struct {
	pattern *regexp.Regexp
	repl    string
}{
	{
		pattern: regexp.MustCompile(`(?i)mongodb(\+srv)?://[^\s"'<>]+`),
		repl:    "mongodb://***",
	},
	{
		pattern: regexp.MustCompile(`(?i)(://)[^/\s"']+@`),
		repl:    "${1}***@",
	},
	{
		pattern: regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._~+/=-]+`),
		repl:    "${1}***",
	},
	{
		pattern: regexp.MustCompile(`\b\d{8,10}:[A-Za-z0-9_-]{20,}\b`),
		repl:    "***:***",
	},
	{
		pattern: regexp.MustCompile(`(?i)(password=)[^&\s"']+`),
		repl:    "${1}***",
	},
	{
		pattern: regexp.MustCompile(`(?i)(refresh_token=)[^&\s"']+`),
		repl:    "${1}***",
	},
	{
		pattern: regexp.MustCompile(`(?i)(access_token=)[^&\s"']+`),
		repl:    "${1}***",
	},
	{
		pattern: regexp.MustCompile(`(?i)(client_secret=)[^&\s"']+`),
		repl:    "${1}***",
	},
	{
		pattern: regexp.MustCompile(`(?i)("access_token"\s*:\s*")([^"]*)(")`),
		repl:    `${1}***${3}`,
	},
	{
		pattern: regexp.MustCompile(`(?i)("refresh_token"\s*:\s*")([^"]*)(")`),
		repl:    `${1}***${3}`,
	},
	{
		pattern: regexp.MustCompile(`(?i)("password"\s*:\s*")([^"]*)(")`),
		repl:    `${1}***${3}`,
	},
	{
		pattern: regexp.MustCompile(`(?i)("client_secret"\s*:\s*")([^"]*)(")`),
		repl:    `${1}***${3}`,
	},
}

func SanitizeString(value string) string {
	if value == "" {
		return value
	}

	result := value
	for _, item := range sanitizePatterns {
		result = item.pattern.ReplaceAllString(result, item.repl)
	}

	return result
}

func SanitizeError(err error) string {
	if err == nil {
		return ""
	}

	return SanitizeString(err.Error())
}
