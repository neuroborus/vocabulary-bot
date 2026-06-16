package telegram

import "testing"

func TestBuildChatAllowlistIncludesAdminAndTargetChannel(t *testing.T) {
	t.Parallel()

	allowlist := BuildChatAllowlist([]int64{200}, 42, -100123)

	if !allowlist.Allows(200) {
		t.Fatal("Allows(200) = false, want true")
	}
	if !allowlist.Allows(42) {
		t.Fatal("Allows(42) = false, want true")
	}
	if !allowlist.Allows(-100123) {
		t.Fatal("Allows(-100123) = false, want true")
	}
	if allowlist.Allows(99) {
		t.Fatal("Allows(99) = true, want false")
	}
}

func TestChatAllowlistAllowsConfiguredIDs(t *testing.T) {
	t.Parallel()

	allowlist := NewChatAllowlist([]int64{42, -100123})

	if !allowlist.Allows(42) {
		t.Fatal("Allows(42) = false, want true")
	}
	if !allowlist.Allows(-100123) {
		t.Fatal("Allows(-100123) = false, want true")
	}
	if allowlist.Allows(99) {
		t.Fatal("Allows(99) = true, want false")
	}
}

func TestIsGroupLikeChat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		chatType string
		want     bool
	}{
		{chatType: "private", want: false},
		{chatType: "group", want: true},
		{chatType: "supergroup", want: true},
		{chatType: "channel", want: true},
	}

	for _, tc := range cases {
		if got := isGroupLikeChat(tc.chatType); got != tc.want {
			t.Fatalf("isGroupLikeChat(%q) = %v, want %v", tc.chatType, got, tc.want)
		}
	}
}
