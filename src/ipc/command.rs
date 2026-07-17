use anyhow::Context;
use serde::de::DeserializeOwned;
use serde_json::json;

use crate::{
    session::SessionError,
    types::{CommandType, ErrorBody, EventType, IpcRequest, IpcResponse, PairCodePayload},
};

use super::{
    contact_commands::{handle_contact_request, is_contact_command},
    media_commands::{handle_media_request, is_media_command},
    message_commands::{handle_message_request, is_message_command},
    presence_commands::{handle_presence_request, is_presence_command},
    server::{IPC_PROTOCOL_VERSION, IpcState},
    subscription::subscribe_response,
};

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
    if is_message_command(request.command) {
        return handle_message_request(request, state).await;
    }

    if is_contact_command(request.command) {
        return handle_contact_request(request, state).await;
    }

    if is_media_command(request.command) {
        return handle_media_request(request, state).await;
    }

    if is_presence_command(request.command) {
        return handle_presence_request(request, state).await;
    }

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

pub(super) fn session_response(
    id: uuid::Uuid,
    result: Result<impl serde::Serialize, SessionError>,
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

pub(super) fn parse_payload<T>(
    id: uuid::Uuid,
    payload: serde_json::Value,
    label: &'static str,
) -> Result<T, IpcResponse>
where
    T: DeserializeOwned,
{
    serde_json::from_value::<T>(payload).map_err(|error| {
        IpcResponse::failure(id, invalid_request(format!("invalid {label}: {error}")))
    })
}

pub(super) fn not_found(message: &str) -> ErrorBody {
    ErrorBody {
        code: "not_found".to_owned(),
        message: message.to_owned(),
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
