use std::{str::FromStr, sync::Arc};

use whatsapp_rust::{Jid, waproto::whatsapp as wa};

use crate::types::{
    OutgoingMessage, SendMessagePayload, SendMessageResultPayload, SendTextPayload, SessionState,
};

use super::{error::SessionError, manager::SessionManager};

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
                    conversation: Some(text),
                    ..Default::default()
                },
            )
            .await
            .map_err(SessionError::SendFailed)?;

        Ok(SendMessageResultPayload {
            message_id: result.message_id,
            chat_jid: result.to.to_string(),
            status: "sent".to_owned(),
        })
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
    let Some((user, _server)) = chat_jid.split_once('@') else {
        return Err(invalid_chat_jid());
    };

    if user.is_empty() {
        return Err(invalid_chat_jid());
    }

    Jid::from_str(chat_jid).map_err(|_error| invalid_chat_jid())
}

fn invalid_chat_jid() -> SessionError {
    SessionError::InvalidRequest("chat_jid must be a valid full WhatsApp JID".to_owned())
}
