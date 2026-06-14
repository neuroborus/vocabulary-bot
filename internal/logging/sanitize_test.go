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
			input: `connect failed: mongodb+srv://vocabulary-bot:SecretPass@cluster0.example.net/?retryWrites=true&w=majority`,
			want:  `connect failed: mongodb://***`,
		},
		{
			name:  "bearer token",
			input: `Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload`,
			want:  `Authorization: Bearer ***`,
		},
		{
			name:  "telegram bot token",
			input: `invalid token 8892229417:AAE1IqPgEwOqQec7fcRK8t-CgmjHh-JvNKE in request`,
			want:  `invalid token ***:*** in request`,
		},
		{
			name:  "query params",
			input: `POST /auth/login?username=reader@example.test&password=SecretPass&refresh_token=abc123`,
			want:  `POST /auth/login?username=reader@example.test&password=***&refresh_token=***`,
		},
		{
			name:  "json tokens",
			input: `{"access_token":"abc123","refresh_token":"def456","password":"SecretPass"}`,
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
