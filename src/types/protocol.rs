use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct IpcRequest {
    pub id: Uuid,
    #[serde(rename = "type")]
    pub command: CommandType,
    #[serde(default)]
    pub payload: Value,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct IpcResponse {
    pub id: Uuid,
    pub ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub payload: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<ErrorBody>,
}

impl IpcResponse {
    pub fn success(id: Uuid, payload: Value) -> Self {
        Self {
            id,
            ok: true,
            payload: Some(payload),
            error: None,
        }
    }

    pub fn failure(id: Uuid, error: ErrorBody) -> Self {
        Self {
            id,
            ok: false,
            payload: None,
            error: Some(error),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ErrorBody {
    pub code: String,
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct DaemonEvent {
    #[serde(rename = "type")]
    pub event: EventType,
    #[serde(default)]
    pub payload: Value,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum CommandType {
    #[serde(rename = "daemon.ping")]
    DaemonPing,
    #[serde(rename = "daemon.version")]
    DaemonVersion,
    #[serde(rename = "daemon.status")]
    DaemonStatus,
    #[serde(rename = "daemon.shutdown")]
    DaemonShutdown,
    #[serde(rename = "session.status")]
    SessionStatus,
    #[serde(rename = "session.connect")]
    SessionConnect,
    #[serde(rename = "session.disconnect")]
    SessionDisconnect,
    #[serde(rename = "session.logout")]
    SessionLogout,
    #[serde(rename = "session.pair_qr")]
    SessionPairQr,
    #[serde(rename = "session.pair_code")]
    SessionPairCode,
    #[serde(rename = "message.send_text")]
    MessageSendText,
    #[serde(rename = "message.send")]
    MessageSend,
    #[serde(rename = "message.react")]
    MessageReact,
    #[serde(rename = "message.edit")]
    MessageEdit,
    #[serde(rename = "message.revoke")]
    MessageRevoke,
    #[serde(rename = "message.mark_read")]
    MessageMarkRead,
    #[serde(rename = "message.list")]
    MessageList,
    #[serde(rename = "message.get")]
    MessageGet,
    #[serde(rename = "event.subscribe")]
    EventSubscribe,
    #[serde(rename = "event.unsubscribe")]
    EventUnsubscribe,
    #[serde(rename = "chat.list")]
    ChatList,
    #[serde(rename = "chat.get")]
    ChatGet,
    #[serde(rename = "chat.archive")]
    ChatArchive,
    #[serde(rename = "chat.pin")]
    ChatPin,
    #[serde(rename = "chat.mute")]
    ChatMute,
    #[serde(rename = "contact.list")]
    ContactList,
    #[serde(rename = "contact.get")]
    ContactGet,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum EventType {
    #[serde(rename = "event.connected")]
    Connected,
    #[serde(rename = "event.disconnected")]
    Disconnected,
    #[serde(rename = "event.logged_out")]
    LoggedOut,
    #[serde(rename = "event.pairing_qr")]
    PairingQr,
    #[serde(rename = "event.pairing_code")]
    PairingCode,
    #[serde(rename = "event.message")]
    Message,
    #[serde(rename = "event.receipt")]
    Receipt,
    #[serde(rename = "event.presence")]
    Presence,
    #[serde(rename = "event.chat_update")]
    ChatUpdate,
    #[serde(rename = "event.group_update")]
    GroupUpdate,
    #[serde(rename = "event.history_sync_progress")]
    HistorySyncProgress,
    #[serde(rename = "event.error")]
    Error,
}
