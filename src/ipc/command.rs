use anyhow::Context;
use serde_json::json;

use crate::{
    session::SessionError,
    types::{
        CommandType, ErrorBody, EventSubscribePayload, EventSubscriptionStatus, EventType,
        IpcRequest, IpcResponse, PairCodePayload,
    },
};

use super::event_filter::{is_supported_event_type, supported_event_types};
use super::server::{IPC_PROTOCOL_VERSION, IpcState};

pub(super) enum CommandOutcome {
    Response(IpcResponse),
    Shutdown(IpcResponse),
    Subscribe {
        response: IpcResponse,
        types: Vec<EventType>,
    },
    Unsubscribe(IpcResponse),
}

pub(super) async fn handle_request(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    match request.command {
        CommandType::DaemonPing => CommandOutcome::Response(IpcResponse::success(
            request.id,
            json!({ "timestamp_unix_seconds": unix_timestamp_seconds() }),
        )),
        CommandType::DaemonVersion => CommandOutcome::Response(IpcResponse::success(
            request.id,
            json!({
                "name": env!("CARGO_PKG_NAME"),
                "version": env!("CARGO_PKG_VERSION"),
                "protocol_version": IPC_PROTOCOL_VERSION,
            }),
        )),
        CommandType::DaemonStatus => match serde_json::to_value(state.daemon_status().await) {
            Ok(payload) => CommandOutcome::Response(IpcResponse::success(request.id, payload)),
            Err(error) => CommandOutcome::Response(IpcResponse::failure(
                request.id,
                internal_error(format!("failed to serialize daemon status: {error}")),
            )),
        },
        CommandType::DaemonShutdown => CommandOutcome::Shutdown(IpcResponse::success(
            request.id,
            json!({ "shutdown": true }),
        )),
        CommandType::SessionStatus | CommandType::SessionPairQr => {
            match serde_json::to_value(state.session_manager.status().await) {
                Ok(payload) => CommandOutcome::Response(IpcResponse::success(request.id, payload)),
                Err(error) => CommandOutcome::Response(IpcResponse::failure(
                    request.id,
                    internal_error(format!("failed to serialize session status: {error}")),
                )),
            }
        }
        CommandType::SessionConnect => session_response(
            request.id,
            state.session_manager.connect().await,
            "failed to serialize session status",
        ),
        CommandType::SessionPairCode => {
            let payload = match serde_json::from_value::<PairCodePayload>(request.payload) {
                Ok(payload) => payload,
                Err(error) => {
                    return CommandOutcome::Response(IpcResponse::failure(
                        request.id,
                        invalid_request(format!("invalid pair-code payload: {error}")),
                    ));
                }
            };

            session_response(
                request.id,
                state.session_manager.pair_code(payload).await,
                "failed to serialize session status",
            )
        }
        CommandType::SessionDisconnect => session_response(
            request.id,
            state.session_manager.disconnect().await,
            "failed to serialize session status",
        ),
        CommandType::SessionLogout => session_response(
            request.id,
            state.session_manager.logout().await,
            "failed to serialize session status",
        ),
        CommandType::EventSubscribe => subscribe_response(request),
        CommandType::EventUnsubscribe => CommandOutcome::Unsubscribe(IpcResponse::success(
            request.id,
            json!({ "subscribed": false }),
        )),
        _ => CommandOutcome::Response(IpcResponse::failure(
            request.id,
            ErrorBody {
                code: "unsupported_command".to_owned(),
                message: "command is not implemented in this milestone".to_owned(),
            },
        )),
    }
}

fn session_response(
    id: uuid::Uuid,
    result: Result<crate::types::SessionStatus, SessionError>,
    serialize_context: &'static str,
) -> CommandOutcome {
    match result {
        Ok(status) => match serde_json::to_value(status).with_context(|| serialize_context) {
            Ok(payload) => CommandOutcome::Response(IpcResponse::success(id, payload)),
            Err(error) => CommandOutcome::Response(IpcResponse::failure(
                id,
                internal_error(error.to_string()),
            )),
        },
        Err(error) => CommandOutcome::Response(IpcResponse::failure(id, session_error(error))),
    }
}

fn subscribe_response(request: IpcRequest) -> CommandOutcome {
    let payload = if request.payload.is_null() {
        EventSubscribePayload { types: Vec::new() }
    } else {
        match serde_json::from_value::<EventSubscribePayload>(request.payload) {
            Ok(payload) => payload,
            Err(error) => {
                return CommandOutcome::Response(IpcResponse::failure(
                    request.id,
                    invalid_request(format!("invalid event subscription payload: {error}")),
                ));
            }
        }
    };

    let selected = if payload.types.is_empty() {
        supported_event_types()
    } else {
        payload.types
    };

    if let Some(event) = selected
        .iter()
        .find(|event| !is_supported_event_type(event))
    {
        return CommandOutcome::Response(IpcResponse::failure(
            request.id,
            invalid_request(format!(
                "event type {event:?} is not supported in this milestone"
            )),
        ));
    }

    let status = EventSubscriptionStatus {
        subscribed: true,
        types: selected.clone(),
    };

    match serde_json::to_value(status) {
        Ok(payload) => CommandOutcome::Subscribe {
            response: IpcResponse::success(request.id, payload),
            types: selected,
        },
        Err(error) => CommandOutcome::Response(IpcResponse::failure(
            request.id,
            internal_error(format!("failed to serialize subscription status: {error}")),
        )),
    }
}

fn unix_timestamp_seconds() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_or(0, |duration| duration.as_secs())
}

fn invalid_request(message: String) -> ErrorBody {
    ErrorBody {
        code: "invalid_request".to_owned(),
        message,
    }
}

fn internal_error(message: String) -> ErrorBody {
    ErrorBody {
        code: "internal_error".to_owned(),
        message,
    }
}

fn session_error(error: SessionError) -> ErrorBody {
    ErrorBody {
        code: error.code().to_owned(),
        message: error.to_string(),
    }
}
