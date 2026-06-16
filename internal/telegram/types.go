package telegram

type Update struct {
	UpdateID      int                `json:"update_id"`
	Message       *Message           `json:"message"`
	CallbackQuery *CallbackQuery     `json:"callback_query"`
	MyChatMember  *ChatMemberUpdated `json:"my_chat_member"`
}

type ChatMemberUpdated struct {
	Chat          Chat       `json:"chat"`
	From          User       `json:"from"`
	Date          int        `json:"date"`
	OldChatMember ChatMember `json:"old_chat_member"`
	NewChatMember ChatMember `json:"new_chat_member"`
}

type ChatMember struct {
	User   User   `json:"user"`
	Status string `json:"status"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type Message struct {
	MessageID      int      `json:"message_id"`
	From           User     `json:"from"`
	Chat           Chat     `json:"chat"`
	Text           string   `json:"text"`
	ReplyToMessage *Message `json:"reply_to_message"`
}

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}
