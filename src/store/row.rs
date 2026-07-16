use diesel::sql_types::{BigInt, Bool, Integer, Nullable, Text};

use crate::{
    store::{MessageRecord, MessageRecordWithReceipt},
    types::LatestReceiptPayload,
};

#[derive(Debug, diesel::QueryableByName)]
pub(super) struct MessageRow {
    #[diesel(sql_type = Text)]
    account_id: String,
    #[diesel(sql_type = Text)]
    chat_jid: String,
    #[diesel(sql_type = Text)]
    sender_jid: String,
    #[diesel(sql_type = Text)]
    message_id: String,
    #[diesel(sql_type = Integer)]
    server_id: i32,
    #[diesel(sql_type = BigInt)]
    timestamp_unix_seconds: i64,
    #[diesel(sql_type = Text)]
    message_type: String,
    #[diesel(sql_type = Nullable<Text>)]
    text: Option<String>,
    #[diesel(sql_type = Bool)]
    is_from_me: bool,
    #[diesel(sql_type = Bool)]
    is_group: bool,
    #[diesel(sql_type = Text)]
    direction: String,
    #[diesel(sql_type = Nullable<Text>)]
    receipt_type: Option<String>,
    #[diesel(sql_type = Nullable<BigInt>)]
    receipt_timestamp_unix_seconds: Option<i64>,
    #[diesel(sql_type = Nullable<Text>)]
    receipt_sender_jid: Option<String>,
}

pub(super) fn row_to_record(row: MessageRow) -> MessageRecordWithReceipt {
    let latest_receipt = row.receipt_type.map(|receipt_type| LatestReceiptPayload {
        receipt_type,
        timestamp_unix_seconds: row.receipt_timestamp_unix_seconds.unwrap_or_default(),
        sender_jid: row.receipt_sender_jid.unwrap_or_default(),
    });

    MessageRecordWithReceipt {
        message: MessageRecord {
            account_id: row.account_id,
            chat_jid: row.chat_jid,
            sender_jid: row.sender_jid,
            message_id: row.message_id,
            server_id: row.server_id,
            timestamp_unix_seconds: row.timestamp_unix_seconds,
            message_type: row.message_type,
            text: row.text,
            is_from_me: row.is_from_me,
            is_group: row.is_group,
            direction: row.direction,
        },
        latest_receipt,
    }
}
