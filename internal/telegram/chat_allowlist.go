package telegram

type ChatAllowlist struct {
	ids map[int64]struct{}
}

func BuildChatAllowlist(configured []int64, alwaysAllow ...int64) ChatAllowlist {
	ids := make([]int64, 0, len(configured)+len(alwaysAllow))
	ids = append(ids, configured...)
	for _, id := range alwaysAllow {
		if id != 0 {
			ids = append(ids, id)
		}
	}

	return NewChatAllowlist(ids)
}

func NewChatAllowlist(ids []int64) ChatAllowlist {
	allowlist := ChatAllowlist{ids: make(map[int64]struct{}, len(ids))}
	for _, id := range ids {
		allowlist.ids[id] = struct{}{}
	}

	return allowlist
}

func (a ChatAllowlist) Allows(chatID int64) bool {
	_, ok := a.ids[chatID]
	return ok
}

func (a ChatAllowlist) Empty() bool {
	return len(a.ids) == 0
}

func isGroupLikeChat(chatType string) bool {
	switch chatType {
	case "group", "supergroup", "channel":
		return true
	default:
		return false
	}
}
