use std::{str::FromStr, sync::Arc};

use whatsapp_rust::{Jid, waproto::whatsapp as wa};

use crate::{
    store::{ListMessagesQuery, MessageRecordInput},
    types::{
        GetMessagePayload, ListMessagesPayload, ListMessagesResultPayload, OutgoingMessage,
        SendMessagePayload, SendMessageResultPayload, SendTextPayload, SessionState,
        StoredMessagePayload,
    },
};

use super::{error::SessionError, manager::SessionManager};

const DEFAULT_LIST_LIMIT: u32 = 50;
const MAX_LIST_LIMIT: u32 = 500;

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
        let client = self.client_for_send().await?;
        let result = client
            .send_message(
                jid,
                wa::Message {
                    conversation: Some(text.clone()),
                    ..Default::default()
                },
            )
            .await
            .map_err(SessionError::SendFailed)?;
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
            status: "sent".to_owned(),
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

    async fn client_for_send(&self) -> Result<Arc<whatsapp_rust::Client>, SessionError> {
        let inner = self.inner.lock().await;
        if inner.state != SessionState::Connected {
            return Err(SessionError::InvalidState(format!(
                "session is {:?}",
                inner.state
            )));
        }

        inner
            .running
            .as_ref()
            .map(|running| Arc::clone(&running.client))
            .ok_or_else(|| SessionError::InvalidState("session is not running".to_owned()))
    }
}

fn parse_full_jid(chat_jid: &str) -> Result<Jid, SessionError> {
    validate_full_jid(chat_jid)?;
    Jid::from_str(chat_jid).map_err(|_error| invalid_chat_jid())
}

fn validate_full_jid(chat_jid: &str) -> Result<(), SessionError> {
    let Some((user, _server)) = chat_jid.split_once('@') else {
        return Err(invalid_chat_jid());
    };

    if user.is_empty() {
        return Err(invalid_chat_jid());
    }

    Jid::from_str(chat_jid)
        .map(|_| ())
        .map_err(|_error| invalid_chat_jid())
}

fn invalid_chat_jid() -> SessionError {
    SessionError::InvalidRequest("chat_jid must be a valid full WhatsApp JID".to_owned())
}

fn is_group_chat(chat_jid: &str) -> bool {
    chat_jid.ends_with("@g.us")
}

fn now_unix_seconds() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_or(0, |duration| duration.as_secs() as i64)
}
