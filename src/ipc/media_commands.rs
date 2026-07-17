use crate::types::{CommandType, IpcRequest, MediaDownloadPayload};

use super::{
    command::{CommandOutcome, parse_payload, session_response},
    server::IpcState,
};

pub(super) fn is_media_command(command: CommandType) -> bool {
    matches!(command, CommandType::MediaDownload)
}

pub(super) async fn handle_media_request(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    match request.command {
        CommandType::MediaDownload => handle_download(request, state).await,
        _ => unreachable!("media command prefiltered"),
    }
}

async fn handle_download(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<MediaDownloadPayload>(
        request.id,
        request.payload,
        "media-download payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };

    session_response(
        request.id,
        state.session_manager.clone().download_media(payload).await,
        "failed to serialize media download result",
    )
}
