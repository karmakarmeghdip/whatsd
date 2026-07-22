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

// EventNotification represents an asynchronous push notification sent to IPC clients.
type EventNotification struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// MessageEventData represents an incoming message payload sent over IPC.
type MessageEventData struct {
	ID        string    `json:"id"`
	Chat      string    `json:"chat"`
	Sender    string    `json:"sender"`
	PushName  string    `json:"push_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	IsFromMe  bool      `json:"is_from_me"`
	IsGroup   bool      `json:"is_group"`
	Text      string    `json:"text"`
}

// QREventData represents a QR code update event.
type QREventData struct {
	Code    string `json:"code,omitempty"`
	Err     string `json:"error,omitempty"`
	Success bool   `json:"success,omitempty"`
}
