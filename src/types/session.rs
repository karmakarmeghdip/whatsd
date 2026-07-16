use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct DaemonStatus {
    pub version: String,
    pub protocol_version: u32,
    pub uptime_seconds: u64,
    pub paths: DaemonPaths,
    pub session: SessionStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct DaemonPaths {
    pub socket: String,
    pub state_dir: String,
    pub database: String,
    pub daemon_database: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SessionStatus {
    pub account_id: String,
    pub state: SessionState,
}

#[derive(Debug, Clone, Deserialize, PartialEq, Eq)]
pub struct PairCodePayload {
    pub phone_number: String,
    #[serde(default)]
    pub custom_code: Option<String>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum SessionState {
    Disconnected,
    Connecting,
    Pairing,
    Connected,
    LoggedOut,
}
