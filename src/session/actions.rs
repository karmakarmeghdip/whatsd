use whatsapp_rust::{RevokeType, waproto::whatsapp as wa};

use crate::{
    store::MessageRecordInput,
    types::{
        ActionStatusPayload, MessageEditPayload, MessageMarkReadPayload, MessageReactPayload,
        MessageRevokePayload,
    },
};

use super::{
    error::SessionError,
    manager::SessionManager,
    messages::{now_unix_seconds, parse_full_jid},
};

impl SessionManager {
    pub async fn react_message(
        &self,
        payload: MessageReactPayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let chat = parse_full_jid(&payload.chat_jid)?;
        let target = self
            .store
            .get_message(
                self.account_id.clone(),
                payload.chat_jid.clone(),
                payload.message_id.clone(),
            )
            .await
            .map_err(SessionError::StoreFailed)?
            .ok_or_else(|| {
                SessionError::InvalidRequest("target message is not in local store".to_owned())
            })?;

        let participant = if target.message.is_group {
            if !target.message.sender_jid.contains('@') {
                return Err(SessionError::InvalidRequest(
                    "stored group message has no WhatsApp sender_jid".to_owned(),
                ));
            }
            Some(target.message.sender_jid.clone())
        } else {
            None
        };

        let reaction = wa::Message {
            reaction_message: Some(wa::message::ReactionMessage {
                key: Some(wa::MessageKey {
                    remote_jid: Some(payload.chat_jid),
                    id: Some(payload.message_id),
                    from_me: Some(target.message.is_from_me),
                    participant,
                }),
                text: Some(payload.emoji),
                sender_timestamp_ms: Some(now_unix_seconds() * 1000),
                ..Default::default()
            }),
            ..Default::default()
        };

        client
            .send_message(chat, reaction)
            .await
            .map_err(SessionError::SendFailed)?;
        Ok(ok_status())
    }

    pub async fn edit_message(
        &self,
        payload: MessageEditPayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let chat = parse_full_jid(&payload.chat_jid)?;
        client
            .edit_message(
                chat,
                payload.message_id.clone(),
                wa::Message {
                    conversation: Some(payload.text.clone()),
                    ..Default::default()
                },
            )
            .await
            .map_err(SessionError::SendFailed)?;
        self.upsert_local_action_message(
            payload.chat_jid,
            payload.message_id,
            Some(payload.text),
            "text",
        )
        .await?;
        Ok(ok_status())
    }

    pub async fn revoke_message(
        &self,
        payload: MessageRevokePayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let chat = parse_full_jid(&payload.chat_jid)?;
        client
            .revoke_message(chat, payload.message_id.clone(), RevokeType::Sender)
            .await
            .map_err(SessionError::SendFailed)?;
        self.upsert_local_action_message(payload.chat_jid, payload.message_id, None, "revoked")
            .await?;
        Ok(ok_status())
    }

    pub async fn mark_read(
        &self,
        payload: MessageMarkReadPayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let chat = parse_full_jid(&payload.chat_jid)?;
        let sender = payload
            .sender_jid
            .as_deref()
            .map(parse_full_jid)
            .transpose()?;
        client
            .mark_as_read(&chat, sender.as_ref(), payload.message_ids)
            .await
            .map_err(SessionError::SendFailed)?;
        Ok(ok_status())
    }

    async fn upsert_local_action_message(
        &self,
        chat_jid: String,
        message_id: String,
        text: Option<String>,
        message_type: &str,
    ) -> Result<(), SessionError> {
        let existing = self
            .store
            .get_message(
                self.account_id.clone(),
                chat_jid.clone(),
                message_id.clone(),
            )
            .await
            .map_err(SessionError::StoreFailed)?;
        let timestamp = existing
            .as_ref()
            .map(|record| record.message.timestamp_unix_seconds)
            .unwrap_or_else(now_unix_seconds);
        let direction = existing
            .as_ref()
            .map(|record| record.message.direction.clone())
            .unwrap_or_else(|| "outgoing".to_owned());
        let is_group = existing
            .as_ref()
            .is_some_and(|record| record.message.is_group)
            || chat_jid.ends_with("@g.us");

        self.store
            .upsert_message(MessageRecordInput {
                account_id: self.account_id.clone(),
                chat_jid,
                sender_jid: self.account_id.clone(),
                message_id,
                server_id: existing
                    .as_ref()
                    .map(|record| record.message.server_id)
                    .unwrap_or_default(),
                timestamp_unix_seconds: timestamp,
                message_type: message_type.to_owned(),
                text,
                is_from_me: true,
                is_group,
                direction,
            })
            .await
            .map_err(SessionError::StoreFailed)
    }
}

fn ok_status() -> ActionStatusPayload {
    ActionStatusPayload {
        status: "ok".to_owned(),
    }
}
