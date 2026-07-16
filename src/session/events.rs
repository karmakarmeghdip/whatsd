use std::sync::Arc;

use tokio::sync::{Mutex, broadcast};
use tracing::debug;
use whatsapp_rust::types::events::{ConnectFailureReason, Event};

use crate::types::{DaemonEvent, EventType, SessionState};

use super::manager::SessionInner;

pub(super) async fn normalize_event(
    event: Arc<Event>,
    inner: Arc<Mutex<SessionInner>>,
    event_tx: broadcast::Sender<DaemonEvent>,
    account_id: String,
    generation: u64,
) {
    let normalized = match &*event {
        Event::PairingQrCode { code, timeout } => {
            state_event(
                &inner,
                generation,
                SessionState::Pairing,
                EventType::PairingQr,
                serde_json::json!({ "code": code, "timeout_seconds": timeout.as_secs() }),
            )
            .await
        }
        Event::PairingCode { code, timeout } => {
            state_event(
                &inner,
                generation,
                SessionState::Pairing,
                EventType::PairingCode,
                serde_json::json!({ "code": code, "timeout_seconds": timeout.as_secs() }),
            )
            .await
        }
        Event::Connected(_) => {
            state_event(
                &inner,
                generation,
                SessionState::Connected,
                EventType::Connected,
                serde_json::json!({}),
            )
            .await
        }
        Event::Disconnected(_) => {
            state_event(
                &inner,
                generation,
                SessionState::Connecting,
                EventType::Disconnected,
                serde_json::json!({}),
            )
            .await
        }
        Event::LoggedOut(info) => {
            state_event(
                &inner,
                generation,
                SessionState::LoggedOut,
                EventType::LoggedOut,
                serde_json::json!({
                    "on_connect": info.on_connect,
                    "reason": connect_failure_reason(&info.reason),
                }),
            )
            .await
        }
        _ => None,
    };

    if let Some((event, mut payload)) = normalized {
        payload["account_id"] = serde_json::Value::String(account_id);
        if event_tx.send(DaemonEvent { event, payload }).is_err() {
            debug!(?event, "no IPC subscribers for session event");
        }
    }
}

async fn state_event(
    inner: &Mutex<SessionInner>,
    generation: u64,
    state: SessionState,
    event: EventType,
    payload: serde_json::Value,
) -> Option<(EventType, serde_json::Value)> {
    let mut inner = inner.lock().await;
    if inner.generation == generation {
        inner.state = state;
        if matches!(state, SessionState::LoggedOut) {
            inner.running = None;
        }
        Some((event, payload))
    } else {
        None
    }
}

fn connect_failure_reason(reason: &ConnectFailureReason) -> &'static str {
    match reason {
        ConnectFailureReason::Generic => "generic",
        ConnectFailureReason::LoggedOut => "logged_out",
        ConnectFailureReason::TempBanned => "temp_banned",
        ConnectFailureReason::MainDeviceGone => "main_device_gone",
        ConnectFailureReason::UnknownLogout => "unknown_logout",
        ConnectFailureReason::ClientOutdated => "client_outdated",
        ConnectFailureReason::BadUserAgent => "bad_user_agent",
        ConnectFailureReason::CatExpired => "cat_expired",
        ConnectFailureReason::CatInvalid => "cat_invalid",
        ConnectFailureReason::NotFound => "not_found",
        ConnectFailureReason::ClientUnknown => "client_unknown",
        ConnectFailureReason::InternalServerError => "internal_server_error",
        ConnectFailureReason::Experimental => "experimental",
        ConnectFailureReason::ServiceUnavailable => "service_unavailable",
        ConnectFailureReason::Unknown(_) => "unknown",
    }
}
