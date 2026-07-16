use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SendTextPayload {
    pub chat_jid: String,
    pub text: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SendMessagePayload {
    pub chat_jid: String,
    pub message: OutgoingMessage,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum OutgoingMessage {
    Text { text: String },
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SendMessageResultPayload {
    pub message_id: String,
    pub chat_jid: String,
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct MessageEventPayload {
    pub account_id: String,
    pub chat_jid: String,
    pub sender_jid: String,
    pub message_id: String,
    pub server_id: i32,
    pub timestamp_unix_seconds: i64,
    pub message_type: String,
    pub text: Option<String>,
    pub is_from_me: bool,
    pub is_group: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ReceiptEventPayload {
    pub account_id: String,
    pub chat_jid: String,
    pub sender_jid: String,
    pub message_ids: Vec<String>,
    pub timestamp_unix_seconds: i64,
    pub receipt_type: String,
    pub is_from_me: bool,
    pub is_group: bool,
}
