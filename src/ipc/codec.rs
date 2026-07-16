use anyhow::{Context, Result};
use serde_json::Value;
use tokio::io::{AsyncWrite, AsyncWriteExt};
use uuid::Uuid;

use crate::types::IpcRequest;

pub(super) enum ParsedRequest {
    Request(IpcRequest),
    Invalid { id: Uuid, message: String },
    Close { reason: String },
}

pub(super) fn parse_request(frame: &[u8]) -> ParsedRequest {
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

pub(super) async fn write_json_line<W, T>(writer: &mut W, value: &T) -> Result<()>
where
    W: AsyncWrite + Unpin,
    T: serde::Serialize,
{
    let response = serde_json::to_string(value).context("failed to serialize IPC frame")?;
    writer
        .write_all(response.as_bytes())
        .await
        .context("failed to write IPC frame")?;
    writer
        .write_all(b"\n")
        .await
        .context("failed to terminate IPC frame")?;
    writer.flush().await.context("failed to flush IPC frame")?;

    Ok(())
}
