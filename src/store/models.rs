use crate::types::{LatestReceiptPayload, StoredMessagePayload};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MessageRecordInput {
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
    pub direction: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ReceiptRecordInput {
    pub account_id: String,
    pub chat_jid: String,
    pub sender_jid: String,
    pub message_ids: Vec<String>,
    pub timestamp_unix_seconds: i64,
    pub receipt_type: String,
    pub is_from_me: bool,
    pub is_group: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ListMessagesQuery {
    pub account_id: String,
    pub chat_jid: String,
    pub limit: i64,
    pub before_timestamp_unix_seconds: Option<i64>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MessageRecord {
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
    pub direction: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MessageRecordWithReceipt {
    pub message: MessageRecord,
    pub latest_receipt: Option<LatestReceiptPayload>,
}

impl From<MessageRecordWithReceipt> for StoredMessagePayload {
    fn from(record: MessageRecordWithReceipt) -> Self {
        Self {
            account_id: record.message.account_id,
            chat_jid: record.message.chat_jid,
            sender_jid: record.message.sender_jid,
            message_id: record.message.message_id,
            server_id: record.message.server_id,
            timestamp_unix_seconds: record.message.timestamp_unix_seconds,
            message_type: record.message.message_type,
            text: record.message.text,
            is_from_me: record.message.is_from_me,
            is_group: record.message.is_group,
            direction: record.message.direction,
            latest_receipt: record.latest_receipt,
        }
    }
}
