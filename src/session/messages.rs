use std::{str::FromStr, time::Duration};

use tokio::time::timeout;
use tracing::{debug, warn};
use whatsapp_rust::{Jid, NodeFilter, SendOptions, waproto::whatsapp as wa};

use crate::{
    store::{ListMessagesQuery, MessageRecordInput},
    types::{
        GetMessagePayload, ListMessagesPayload, ListMessagesResultPayload, OutgoingMessage,
        SendMessagePayload, SendMessageResultPayload, SendTextPayload, StoredMessagePayload,
    },
};

use super::{error::SessionError, manager::SessionManager};

const DEFAULT_LIST_LIMIT: u32 = 50;
const MAX_LIST_LIMIT: u32 = 500;
const SEND_ACK_TIMEOUT: Duration = Duration::from_secs(10);

impl SessionManager {
    pub async fn send_text(
        &self,
        payload: SendTextPayload,
    ) -> Result<SendMessageResultPayload, SessionError> {
        self.send_text_to(payload.chat_jid, payload.text).await
    }

    pub async fn send_message(
        &self,
        payload: SendMessagePayload,
    ) -> Result<SendMessageResultPayload, SessionError> {
        match payload.message {
            OutgoingMessage::Text { text } => self.send_text_to(payload.chat_jid, text).await,
        }
    }

    async fn send_text_to(
        &self,
        chat_jid: String,
        text: String,
    ) -> Result<SendMessageResultPayload, SessionError> {
        let jid = parse_full_jid(&chat_jid)?;
        let client = self.client_for_connected().await?;
        let message_id = client.generate_message_id().await;
        let ack_waiter = client.wait_for_node(NodeFilter::tag("ack").attr("id", &message_id));

        debug!(
            account_id = %self.account_id,
            target_kind = jid_kind(&chat_jid),
            message_id = %message_id,
            text_bytes = text.len(),
            "submitting outgoing text message"
        );

        let result = client
            .send_message_with_options(
                jid,
                text_message(&text),
                SendOptions {
                    message_id: Some(message_id.clone()),
                    ..Default::default()
                },
            )
            .await
            .map_err(SessionError::SendFailed)?;
        wait_for_send_ack(
            ack_waiter,
            &message_id,
            &self.account_id,
            jid_kind(&chat_jid),
        )
        .await?;

        let chat_jid = result.to.to_string();

        self.store
            .upsert_message(MessageRecordInput {
                account_id: self.account_id.clone(),
                chat_jid: chat_jid.clone(),
                sender_jid: self.account_id.clone(),
                message_id: result.message_id.clone(),
                server_id: 0,
                timestamp_unix_seconds: now_unix_seconds(),
                message_type: "text".to_owned(),
                text: Some(text),
                is_from_me: true,
                is_group: is_group_chat(&chat_jid),
                direction: "outgoing".to_owned(),
            })
            .await
            .map_err(SessionError::StoreFailed)?;

        Ok(SendMessageResultPayload {
            message_id: result.message_id,
            chat_jid,
            status: "server_ack".to_owned(),
        })
    }

    pub async fn list_messages(
        &self,
        payload: ListMessagesPayload,
    ) -> Result<ListMessagesResultPayload, SessionError> {
        validate_full_jid(&payload.chat_jid)?;
        let limit = payload
            .limit
            .unwrap_or(DEFAULT_LIST_LIMIT)
            .clamp(1, MAX_LIST_LIMIT) as i64;
        let messages = self
            .store
            .list_messages(ListMessagesQuery {
                account_id: self.account_id.clone(),
                chat_jid: payload.chat_jid,
                limit,
                before_timestamp_unix_seconds: payload.before_timestamp_unix_seconds,
            })
            .await
            .map_err(SessionError::StoreFailed)?
            .into_iter()
            .map(StoredMessagePayload::from)
            .collect();

        Ok(ListMessagesResultPayload { messages })
    }

    pub async fn get_message(
        &self,
        payload: GetMessagePayload,
    ) -> Result<Option<StoredMessagePayload>, SessionError> {
        validate_full_jid(&payload.chat_jid)?;
        self.store
            .get_message(
                self.account_id.clone(),
                payload.chat_jid,
                payload.message_id,
            )
            .await
            .map_err(SessionError::StoreFailed)
            .map(|message| message.map(StoredMessagePayload::from))
    }
}

async fn wait_for_send_ack<E>(
    ack_waiter: impl std::future::Future<
        Output = Result<std::sync::Arc<whatsapp_rust::OwnedNodeRef>, E>,
    >,
    message_id: &str,
    account_id: &str,
    target_kind: &str,
) -> Result<(), SessionError>
where
    E: std::fmt::Debug,
{
    match timeout(SEND_ACK_TIMEOUT, ack_waiter).await {
        Ok(Ok(node)) => {
            let ack = node.get();
            if let Some(error) = ack
                .get_attr("error")
                .map(|value| value.as_str().into_owned())
            {
                warn!(
                    account_id = %account_id,
                    target_kind,
                    message_id,
                    error_code = %error,
                    "WhatsApp rejected outgoing message"
                );
                return Err(SessionError::SendFailed(anyhow::anyhow!(
                    "WhatsApp server rejected message with ack error {error}"
                )));
            }

            debug!(
                account_id = %account_id,
                target_kind,
                message_id,
                has_phash = ack.get_attr("phash").is_some(),
                "WhatsApp acknowledged outgoing message"
            );
            Ok(())
        }
        Ok(Err(_closed)) => {
            warn!(
                account_id = %account_id,
                target_kind,
                message_id,
                "outgoing message ack waiter closed before receiving server ack"
            );
            Err(SessionError::SendFailed(anyhow::anyhow!(
                "ack waiter closed before WhatsApp server acknowledged message"
            )))
        }
        Err(_elapsed) => {
            warn!(
                account_id = %account_id,
                target_kind,
                message_id,
                timeout_seconds = SEND_ACK_TIMEOUT.as_secs(),
                "timed out waiting for outgoing message server ack"
            );
            Err(SessionError::SendFailed(anyhow::anyhow!(
                "timed out waiting for WhatsApp server ack"
            )))
        }
    }
}

fn text_message(text: &str) -> wa::Message {
    wa::Message {
        extended_text_message: Some(Box::new(wa::message::ExtendedTextMessage {
            text: Some(text.to_owned()),
            ..Default::default()
        })),
        ..Default::default()
    }
}

pub(super) fn parse_full_jid(chat_jid: &str) -> Result<Jid, SessionError> {
    parse_full_jid_field(chat_jid, "chat_jid")
}

pub(super) fn parse_full_jid_field(jid: &str, field_name: &str) -> Result<Jid, SessionError> {
    validate_full_jid_field(jid, field_name)?;
    Jid::from_str(jid).map_err(|_error| invalid_jid(field_name))
}

pub(super) fn validate_full_jid(chat_jid: &str) -> Result<(), SessionError> {
    validate_full_jid_field(chat_jid, "chat_jid")
}

fn validate_full_jid_field(jid: &str, field_name: &str) -> Result<(), SessionError> {
    let Some((user, _server)) = jid.split_once('@') else {
        return Err(invalid_jid(field_name));
    };

    if user.is_empty() {
        return Err(invalid_jid(field_name));
    }

    Jid::from_str(jid)
        .map(|_| ())
        .map_err(|_error| invalid_jid(field_name))
}

fn invalid_jid(field_name: &str) -> SessionError {
    SessionError::InvalidRequest(format!("{field_name} must be a valid full WhatsApp JID"))
}

fn is_group_chat(chat_jid: &str) -> bool {
    chat_jid.ends_with("@g.us")
}

fn jid_kind(jid: &str) -> &'static str {
    if jid.ends_with("@g.us") {
        "group"
    } else if jid.ends_with("@lid") {
        "lid"
    } else if jid.ends_with("@s.whatsapp.net") {
        "pn"
    } else {
        "other"
    }
}

pub(super) fn now_unix_seconds() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_or(0, |duration| duration.as_secs() as i64)
}
