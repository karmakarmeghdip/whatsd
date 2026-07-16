use std::{collections::HashSet, sync::Arc};

use anyhow::{Context, Result};
use tokio::{
    io::{AsyncBufReadExt, AsyncWrite, BufReader},
    net::{UnixStream, unix::OwnedReadHalf},
    sync::{broadcast, watch},
};
use tracing::warn;

use crate::types::{DaemonEvent, ErrorBody, EventType, IpcResponse};

use super::{
    codec::{ParsedRequest, parse_request, write_json_line},
    command::{CommandOutcome, handle_request},
    server::IpcState,
};

const MAX_FRAME_BYTES: usize = 64 * 1024;

pub(super) async fn handle_connection(
    stream: UnixStream,
    state: Arc<IpcState>,
    shutdown_tx: watch::Sender<bool>,
) -> Result<()> {
    let (read_half, mut write_half) = stream.into_split();
    let mut reader = BufReader::new(read_half);
    let mut frame = Vec::new();
    let mut subscription: Option<EventSubscription> = None;

    loop {
        frame.clear();

        if let Some(active) = subscription.as_mut() {
            tokio::select! {
                read_result = read_frame(&mut reader, &mut frame) => {
                    if !handle_frame(read_result?, &frame, &state, &shutdown_tx, &mut write_half, &mut subscription).await? {
                        break;
                    }
                }
                event_result = active.receiver.recv() => {
                    if !handle_event(event_result, active, &mut write_half).await? {
                        break;
                    }
                }
            }
        } else if !handle_frame(
            read_frame(&mut reader, &mut frame).await?,
            &frame,
            &state,
            &shutdown_tx,
            &mut write_half,
            &mut subscription,
        )
        .await?
        {
            break;
        }
    }

    Ok(())
}

async fn read_frame(reader: &mut BufReader<OwnedReadHalf>, frame: &mut Vec<u8>) -> Result<usize> {
    reader
        .read_until(b'\n', frame)
        .await
        .context("failed to read IPC frame")
}

async fn handle_frame<W>(
    bytes_read: usize,
    frame: &[u8],
    state: &Arc<IpcState>,
    shutdown_tx: &watch::Sender<bool>,
    writer: &mut W,
    subscription: &mut Option<EventSubscription>,
) -> Result<bool>
where
    W: AsyncWrite + Unpin,
{
    if bytes_read == 0 {
        return Ok(false);
    }

    if frame.len() > MAX_FRAME_BYTES {
        warn!(
            bytes = frame.len(),
            "closing IPC client after oversized frame"
        );
        return Ok(false);
    }

    let request = match parse_request(frame) {
        ParsedRequest::Request(request) => request,
        ParsedRequest::Invalid { id, message } => {
            let response = IpcResponse::failure(id, invalid_request(message));
            write_json_line(writer, &response).await?;
            return Ok(true);
        }
        ParsedRequest::Close { reason } => {
            warn!(%reason, "closing IPC client after invalid frame without request id");
            return Ok(false);
        }
    };

    match handle_request(request, state).await {
        CommandOutcome::Response(response) => write_json_line(writer, &response).await?,
        CommandOutcome::Shutdown(response) => {
            write_json_line(writer, &response).await?;
            let _ = shutdown_tx.send(true);
            return Ok(false);
        }
        CommandOutcome::Subscribe { response, types } => {
            *subscription = Some(EventSubscription {
                receiver: state.session_manager.subscribe(),
                types: types.into_iter().collect(),
            });
            write_json_line(writer, &response).await?;
        }
        CommandOutcome::Unsubscribe(response) => {
            *subscription = None;
            write_json_line(writer, &response).await?;
        }
    }

    Ok(true)
}

async fn handle_event<W>(
    result: Result<DaemonEvent, broadcast::error::RecvError>,
    subscription: &EventSubscription,
    writer: &mut W,
) -> Result<bool>
where
    W: AsyncWrite + Unpin,
{
    match result {
        Ok(event) => {
            if subscription.types.contains(&event.event) {
                write_json_line(writer, &event).await?;
            }
            Ok(true)
        }
        Err(broadcast::error::RecvError::Lagged(skipped)) => {
            warn!(
                skipped,
                "closing IPC client after event subscription lagged"
            );
            Ok(false)
        }
        Err(broadcast::error::RecvError::Closed) => Ok(false),
    }
}

fn invalid_request(message: String) -> ErrorBody {
    ErrorBody {
        code: "invalid_request".to_owned(),
        message,
    }
}

struct EventSubscription {
    receiver: broadcast::Receiver<DaemonEvent>,
    types: HashSet<EventType>,
}
