use std::sync::Arc;

use tokio::sync::{Mutex, broadcast};
use tracing::debug;
use whatsapp_rust::types::events::{ConnectFailureReason, Event};

use crate::{
    store::Store,
    types::{DaemonEvent, EventType, MessageEventPayload, ReceiptEventPayload, SessionState},
};

use super::{
    manager::SessionInner,
    normalize_message::{
        message_event_payload, message_record_input, receipt_event_payload, receipt_record_input,
    },
};

pub(super) async fn normalize_event(
    event: Arc<Event>,
    inner: Arc<Mutex<SessionInner>>,
    event_tx: broadcast::Sender<DaemonEvent>,
    store: Store,
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
        Event::Message(message, info) => {
            persist_message_event(&store, message_event_payload(&account_id, message, info)).await
        }
        Event::Receipt(receipt) => {
            persist_receipt_event(&store, receipt_event_payload(&account_id, receipt)).await
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

async fn persist_message_event(
    store: &Store,
    payload: MessageEventPayload,
) -> Option<(EventType, serde_json::Value)> {
    match store
        .upsert_message(message_record_input(&payload, "inbound"))
        .await
    {
        Ok(()) => serialize_event(EventType::Message, payload),
        Err(error) => persistence_error_event(
            &payload.account_id,
            "event.message",
            Some(&payload.chat_jid),
            Some(&payload.message_id),
            error,
        ),
    }
}

async fn persist_receipt_event(
    store: &Store,
    payload: ReceiptEventPayload,
) -> Option<(EventType, serde_json::Value)> {
    match store.upsert_receipt(receipt_record_input(&payload)).await {
        Ok(()) => serialize_event(EventType::Receipt, payload),
        Err(error) => persistence_error_event(
            &payload.account_id,
            "event.receipt",
            Some(&payload.chat_jid),
            payload.message_ids.first().map(String::as_str),
            error,
        ),
    }
}

fn persistence_error_event(
    account_id: &str,
    source_event: &str,
    chat_jid: Option<&str>,
    message_id: Option<&str>,
    error: anyhow::Error,
) -> Option<(EventType, serde_json::Value)> {
    debug!(%error, source_event, "failed to persist inbound event");
    Some((
        EventType::Error,
        serde_json::json!({
            "account_id": account_id,
            "code": "store_write_failed",
            "message": "failed to persist inbound event",
            "source_event": source_event,
            "chat_jid": chat_jid,
            "message_id": message_id,
        }),
    ))
}

fn serialize_event<T: serde::Serialize>(
    event: EventType,
    payload: T,
) -> Option<(EventType, serde_json::Value)> {
    match serde_json::to_value(payload) {
        Ok(payload) => Some((event, payload)),
        Err(error) => {
            debug!(%error, ?event, "failed to serialize normalized session event");
            None
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
