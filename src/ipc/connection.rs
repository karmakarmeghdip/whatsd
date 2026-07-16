use std::{
    sync::Arc,
    time::{SystemTime, UNIX_EPOCH},
};

use anyhow::{Context, Result};
use serde_json::{Value, json};
use tokio::{
    io::{AsyncBufReadExt, AsyncWriteExt, BufReader},
    net::UnixStream,
    sync::watch,
};
use tracing::warn;
use uuid::Uuid;

use crate::types::{CommandType, ErrorBody, IpcRequest, IpcResponse};

use super::server::{IPC_PROTOCOL_VERSION, IpcState};

const MAX_FRAME_BYTES: usize = 64 * 1024;

pub(super) async fn handle_connection(
    stream: UnixStream,
    state: Arc<IpcState>,
    shutdown_tx: watch::Sender<bool>,
) -> Result<()> {
    let mut reader = BufReader::new(stream);
    let mut frame = Vec::new();

    loop {
        frame.clear();
        let bytes_read = reader
            .read_until(b'\n', &mut frame)
            .await
            .context("failed to read IPC frame")?;

        if bytes_read == 0 {
            break;
        }

        if frame.len() > MAX_FRAME_BYTES {
            warn!(
                bytes = frame.len(),
                "closing IPC client after oversized frame"
            );
            break;
        }

        let request = match parse_request(&frame) {
            ParsedRequest::Request(request) => request,
            ParsedRequest::Invalid { id, message } => {
                let response = IpcResponse::failure(id, invalid_request(message));
                write_response(reader.get_mut(), &response).await?;
                continue;
            }
            ParsedRequest::Close { reason } => {
                warn!(%reason, "closing IPC client after invalid frame without request id");
                break;
            }
        };

        let request_id = request.id;
        let (response, should_shutdown) = match handle_request(request, &state) {
            Ok(result) => result,
            Err(error) => {
                warn!(%error, "IPC request failed");
                (
                    IpcResponse::failure(request_id, internal_error("request failed")),
                    false,
                )
            }
        };

        write_response(reader.get_mut(), &response).await?;

        if should_shutdown {
            let _ = shutdown_tx.send(true);
            break;
        }
    }

    Ok(())
}

fn parse_request(frame: &[u8]) -> ParsedRequest {
    let value: Value = match serde_json::from_slice(frame) {
        Ok(value) => value,
        Err(error) => {
            return ParsedRequest::Close {
                reason: format!("malformed JSON: {error}"),
            };
        }
    };

    let Some(id_value) = value.get("id") else {
        return ParsedRequest::Close {
            reason: "missing request id".to_owned(),
        };
    };

    let id: Uuid = match serde_json::from_value(id_value.clone()) {
        Ok(id) => id,
        Err(error) => {
            return ParsedRequest::Close {
                reason: format!("invalid request id: {error}"),
            };
        }
    };

    match serde_json::from_value(value) {
        Ok(request) => ParsedRequest::Request(request),
        Err(error) => ParsedRequest::Invalid {
            id,
            message: format!("invalid request: {error}"),
        },
    }
}

fn handle_request(request: IpcRequest, state: &IpcState) -> Result<(IpcResponse, bool)> {
    let response = match request.command {
        CommandType::DaemonPing => IpcResponse::success(
            request.id,
            json!({ "timestamp_unix_seconds": unix_timestamp_seconds()? }),
        ),
        CommandType::DaemonVersion => IpcResponse::success(
            request.id,
            json!({
                "name": env!("CARGO_PKG_NAME"),
                "version": env!("CARGO_PKG_VERSION"),
                "protocol_version": IPC_PROTOCOL_VERSION,
            }),
        ),
        CommandType::DaemonStatus => IpcResponse::success(
            request.id,
            serde_json::to_value(state.daemon_status())
                .context("failed to serialize daemon status")?,
        ),
        CommandType::DaemonShutdown => {
            return Ok((
                IpcResponse::success(request.id, json!({ "shutdown": true })),
                true,
            ));
        }
        _ => IpcResponse::failure(
            request.id,
            ErrorBody {
                code: "unsupported_command".to_owned(),
                message: "command is not implemented in this milestone".to_owned(),
            },
        ),
    };

    Ok((response, false))
}

async fn write_response(stream: &mut UnixStream, response: &IpcResponse) -> Result<()> {
    let response = serde_json::to_string(response).context("failed to serialize IPC response")?;
    stream
        .write_all(response.as_bytes())
        .await
        .context("failed to write IPC response")?;
    stream
        .write_all(b"\n")
        .await
        .context("failed to terminate IPC response")?;
    stream
        .flush()
        .await
        .context("failed to flush IPC response")?;

    Ok(())
}

fn unix_timestamp_seconds() -> Result<u64> {
    Ok(SystemTime::now()
       .duration_since(UNIX_EPOCH)
       .context("system clock is before the Unix epoch")?
       .as_secs())
}

fn invalid_request(message: String) -> ErrorBody {
    ErrorBody {
        code: "invalid_request".to_owned(),
        message,
    }
}

fn internal_error(message: &str) -> ErrorBody {
    ErrorBody {
        code: "internal_error".to_owned(),
        message: message.to_owned(),
    }
}

enum ParsedRequest {
    Request(IpcRequest),
    Invalid { id: Uuid, message: String },
    Close { reason: String },
}
