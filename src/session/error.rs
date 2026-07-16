use thiserror::Error;

#[derive(Debug, Error)]
pub enum SessionError {
    #[error("invalid session request: {0}")]
    InvalidRequest(String),
    #[error("invalid session state: {0}")]
    InvalidState(String),
    #[error("failed to start WhatsApp session")]
    StartFailed(#[source] anyhow::Error),
    #[error("failed to stop WhatsApp session")]
    StopFailed(#[source] anyhow::Error),
}

impl SessionError {
    pub fn code(&self) -> &'static str {
        match self {
            Self::InvalidRequest(_) => "invalid_request",
            Self::InvalidState(_) => "invalid_state",
            Self::StartFailed(_) => "session_start_failed",
            Self::StopFailed(_) => "session_stop_failed",
        }
    }
}
