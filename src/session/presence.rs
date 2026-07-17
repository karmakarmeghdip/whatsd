use whatsapp_rust::{ChatStateType, PresenceStatus};

use crate::types::{
    ActionStatusPayload, ChatStatePayload, ChatStateSendPayload, PresenceSetPayload,
    PresenceStatusPayload, PresenceSubscriptionPayload,
};

use super::{error::SessionError, manager::SessionManager, messages::parse_full_jid};

impl SessionManager {
    pub async fn set_presence(
        &self,
        payload: PresenceSetPayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let status = match payload.status {
            PresenceStatusPayload::Available => PresenceStatus::Available,
            PresenceStatusPayload::Unavailable => PresenceStatus::Unavailable,
        };
        client
            .presence()
            .set(status)
            .await
            .map_err(|error| SessionError::SendFailed(error.into()))?;
        Ok(ok_status())
    }

    pub async fn subscribe_presence(
        &self,
        payload: PresenceSubscriptionPayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let jid = parse_full_jid(&payload.jid)?;
        client
            .presence()
            .subscribe(&jid)
            .await
            .map_err(SessionError::SendFailed)?;
        Ok(ok_status())
    }

    pub async fn unsubscribe_presence(
        &self,
        payload: PresenceSubscriptionPayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let jid = parse_full_jid(&payload.jid)?;
        client
            .presence()
            .unsubscribe(&jid)
            .await
            .map_err(SessionError::SendFailed)?;
        Ok(ok_status())
    }

    pub async fn send_chat_state(
        &self,
        payload: ChatStateSendPayload,
    ) -> Result<ActionStatusPayload, SessionError> {
        let client = self.client_for_connected().await?;
        let jid = parse_full_jid(&payload.chat_jid)?;
        let state = match payload.state {
            ChatStatePayload::Composing => ChatStateType::Composing,
            ChatStatePayload::Recording => ChatStateType::Recording,
            ChatStatePayload::Paused => ChatStateType::Paused,
        };
        client
            .chatstate()
            .send(&jid, state)
            .await
            .map_err(|error| SessionError::SendFailed(error.into()))?;
        Ok(ok_status())
    }
}

fn ok_status() -> ActionStatusPayload {
    ActionStatusPayload {
        status: "ok".to_owned(),
    }
}
