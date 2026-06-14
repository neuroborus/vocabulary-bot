package logging

import "testing"

func TestSanitizeStringRedactsSensitiveValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "mongodb uri",
			input: `connect failed: mongodb+srv://test-user:FAKE_PASSWORD_FOR_TEST_ONLY@cluster0.example.test/?retryWrites=true&w=majority`,
			want:  `connect failed: mongodb://***`,
		},
		{
			name:  "bearer token",
			input: `Authorization: Bearer FAKE_BEARER_TOKEN_FOR_TEST_ONLY`,
			want:  `Authorization: Bearer ***`,
		},
		{
			name:  "telegram bot token",
			input: `invalid token 1234567890:FAKE_TELEGRAM_BOT_TOKEN_FOR_TEST_ONLY in request`,
			want:  `invalid token ***:*** in request`,
		},
		{
			name:  "query params",
			input: `POST /auth/login?username=test-user@example.test&password=FAKE_PASSWORD_FOR_TEST_ONLY&refresh_token=FAKE_REFRESH_TOKEN_FOR_TEST_ONLY`,
			want:  `POST /auth/login?username=test-user@example.test&password=***&refresh_token=***`,
		},
		{
			name:  "json tokens",
			input: `{"access_token":"FAKE_ACCESS_TOKEN_FOR_TEST_ONLY","refresh_token":"FAKE_REFRESH_TOKEN_FOR_TEST_ONLY","password":"FAKE_PASSWORD_FOR_TEST_ONLY"}`,
			want:  `{"access_token":"***","refresh_token":"***","password":"***"}`,
		},
		{
			name:  "plain text unchanged",
			input: `json: cannot unmarshal number into Go struct field BookMetadata.items.metadata.year of type string`,
			want:  `json: cannot unmarshal number into Go struct field BookMetadata.items.metadata.year of type string`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := SanitizeString(test.input); got != test.want {
				t.Fatalf("SanitizeString() = %q, want %q", got, test.want)
			}
		})
	}
}
