package types

import "time"

// Request represents a incoming JSON-RPC request from an IPC client.
type Request struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

// Response represents a JSON-RPC response to an IPC client.
type Response struct {
	ID     string `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// StatusResult contains daemon session state details.
type StatusResult struct {
	Connected bool   `json:"connected"`
	LoggedIn  bool   `json:"logged_in"`
	JID       string `json:"jid,omitempty"`
	PushName  string `json:"push_name,omitempty"`
}

// PairPhoneParams holds arguments for initiating phone pairing.
type PairPhoneParams struct {
	Phone string `json:"phone"`
}

// PairPhoneResult contains the generated 8-character pairing code.
type PairPhoneResult struct {
	Code string `json:"code"`
}

// SendMessageParams holds parameters to send a text message.
type SendMessageParams struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

// SendMessageResult contains confirmation of a sent message.
type SendMessageResult struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

// SendMediaParams holds parameters to send a media message.
type SendMediaParams struct {
	To        string `json:"to"`
	MediaType string `json:"media_type"`
	FilePath  string `json:"file_path"`
	Caption   string `json:"caption,omitempty"`
	FileName  string `json:"file_name,omitempty"`
}

// ContactItem represents a contact directory entry.
type ContactItem struct {
	JID          string `json:"jid"`
	FirstName    string `json:"first_name,omitempty"`
	FullName     string `json:"full_name,omitempty"`
	PushName     string `json:"push_name,omitempty"`
	BusinessName string `json:"business_name,omitempty"`
}

// EventNotification represents an asynchronous push notification sent to IPC clients.
type EventNotification struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// MessageEventData represents an incoming message payload sent over IPC.
type MessageEventData struct {
	ID         string    `json:"id"`
	Chat       string    `json:"chat"`
	Sender     string    `json:"sender"`
	PushName   string    `json:"push_name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	IsFromMe   bool      `json:"is_from_me"`
	IsGroup    bool      `json:"is_group"`
	Text       string    `json:"text"`
	MediaType  string    `json:"media_type,omitempty"`
	Caption    string    `json:"caption,omitempty"`
	FileName   string    `json:"file_name,omitempty"`
	MimeType   string    `json:"mime_type,omitempty"`
	FileLength uint64    `json:"file_length,omitempty"`
}

// ChatItem represents a conversation summary for UI or CLI clients.
type ChatItem struct {
	JID                  string    `json:"jid"`
	Name                 string    `json:"name"`
	LastMessageID        string    `json:"last_message_id,omitempty"`
	LastMessageText      string    `json:"last_message_text,omitempty"`
	LastMessageTimestamp time.Time `json:"last_message_timestamp,omitempty"`
	UnreadCount          int       `json:"unread_count"`
	IsMuted              bool      `json:"is_muted"`
	IsPinned             bool      `json:"is_pinned"`
	IsArchived           bool      `json:"is_archived"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// MessageItem represents a stored historical or active message.
type MessageItem struct {
	ID         string    `json:"id"`
	Chat       string    `json:"chat"`
	Sender     string    `json:"sender"`
	SenderName string    `json:"sender_name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	IsFromMe   bool      `json:"is_from_me"`
	IsGroup    bool      `json:"is_group"`
	Text       string    `json:"text"`
	MediaType  string    `json:"media_type,omitempty"`
	MediaPath  string    `json:"media_path,omitempty"`
	Caption    string    `json:"caption,omitempty"`
	FileName   string    `json:"file_name,omitempty"`
	MimeType   string    `json:"mime_type,omitempty"`
	FileLength uint64    `json:"file_length,omitempty"`
	RawMessage []byte    `json:"-"`
	ReplyToID  string    `json:"reply_to_id,omitempty"`
	Status     string    `json:"status"` // "pending", "sent", "delivered", "read", "received"
	IsEdited   bool      `json:"is_edited"`
	IsRevoked  bool      `json:"is_revoked"`
}

// GetChatsParams holds arguments for querying active chats.
type GetChatsParams struct {
	Limit int `json:"limit,omitempty"`
}

// GetMessagesParams holds arguments for querying message history.
type GetMessagesParams struct {
	Chat     string `json:"chat"`
	Limit    int    `json:"limit,omitempty"`
	BeforeID string `json:"before_id,omitempty"`
}

// MarkReadParams holds arguments for marking a chat as read.
type MarkReadParams struct {
	Chat string `json:"chat"`
}

// HistorySyncProgressEventData represents history sync progress event payload.
type HistorySyncProgressEventData struct {
	ProgressPercent int    `json:"progress_percent"`
	SyncType        string `json:"sync_type"`
}

// QREventData represents a QR code update event.
type QREventData struct {
	Code    string `json:"code,omitempty"`
	Err     string `json:"error,omitempty"`
	Success bool   `json:"success,omitempty"`
}
