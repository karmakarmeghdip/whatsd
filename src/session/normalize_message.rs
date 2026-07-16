use std::sync::Arc;

use whatsapp_rust::{
    proto_helpers::MessageExt,
    types::{events::Receipt, message::MessageInfo, presence::ReceiptType},
    waproto::whatsapp as wa,
};

use crate::{
    store::{MessageRecordInput, ReceiptRecordInput},
    types::{MessageEventPayload, ReceiptEventPayload},
};

pub(super) fn message_event_payload(
    account_id: &str,
    message: &Arc<wa::Message>,
    info: &Arc<MessageInfo>,
) -> MessageEventPayload {
    MessageEventPayload {
        account_id: account_id.to_owned(),
        chat_jid: info.source.chat.to_string(),
        sender_jid: info.source.sender.to_string(),
        message_id: info.id.clone(),
        server_id: info.server_id,
        timestamp_unix_seconds: info.timestamp.timestamp(),
        message_type: info.r#type.clone(),
        text: message.text_content().map(str::to_owned),
        is_from_me: info.source.is_from_me,
        is_group: info.source.is_group,
    }
}

pub(super) fn message_record_input(
    payload: &MessageEventPayload,
    direction: &str,
) -> MessageRecordInput {
    MessageRecordInput {
        account_id: payload.account_id.clone(),
        chat_jid: payload.chat_jid.clone(),
        sender_jid: payload.sender_jid.clone(),
        message_id: payload.message_id.clone(),
        server_id: payload.server_id,
        timestamp_unix_seconds: payload.timestamp_unix_seconds,
        message_type: payload.message_type.clone(),
        text: payload.text.clone(),
        is_from_me: payload.is_from_me,
        is_group: payload.is_group,
        direction: direction.to_owned(),
    }
}

pub(super) fn receipt_event_payload(account_id: &str, receipt: &Receipt) -> ReceiptEventPayload {
    ReceiptEventPayload {
        account_id: account_id.to_owned(),
        chat_jid: receipt.source.chat.to_string(),
        sender_jid: receipt.source.sender.to_string(),
        message_ids: receipt.message_ids.clone(),
        timestamp_unix_seconds: receipt.timestamp.timestamp(),
        receipt_type: receipt_type_name(&receipt.r#type),
        is_from_me: receipt.source.is_from_me,
        is_group: receipt.source.is_group,
    }
}

pub(super) fn receipt_record_input(payload: &ReceiptEventPayload) -> ReceiptRecordInput {
    ReceiptRecordInput {
        account_id: payload.account_id.clone(),
        chat_jid: payload.chat_jid.clone(),
        sender_jid: payload.sender_jid.clone(),
        message_ids: payload.message_ids.clone(),
        timestamp_unix_seconds: payload.timestamp_unix_seconds,
        receipt_type: payload.receipt_type.clone(),
        is_from_me: payload.is_from_me,
        is_group: payload.is_group,
    }
}

fn receipt_type_name(receipt_type: &ReceiptType) -> String {
    match receipt_type {
        ReceiptType::Delivered => "delivered".to_owned(),
        ReceiptType::Sender => "sender".to_owned(),
        ReceiptType::Retry => "retry".to_owned(),
        ReceiptType::EncRekeyRetry => "enc_rekey_retry".to_owned(),
        ReceiptType::Read => "read".to_owned(),
        ReceiptType::ReadSelf => "read_self".to_owned(),
        ReceiptType::Played => "played".to_owned(),
        ReceiptType::PlayedSelf => "played_self".to_owned(),
        ReceiptType::ServerError => "server_error".to_owned(),
        ReceiptType::Inactive => "inactive".to_owned(),
        ReceiptType::PeerMsg => "peer_msg".to_owned(),
        ReceiptType::HistorySync => "history_sync".to_owned(),
        ReceiptType::Other(value) => value.clone(),
    }
}
