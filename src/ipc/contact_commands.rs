use crate::types::{CommandType, ContactGetPayload, ContactProfilePicturePayload, IpcRequest};

use super::{
    command::{CommandOutcome, not_found, parse_payload, session_response},
    server::IpcState,
};

pub(super) fn is_contact_command(command: CommandType) -> bool {
    matches!(
        command,
        CommandType::ContactGet | CommandType::ContactProfilePicture
    )
}

pub(super) async fn handle_contact_request(
    request: IpcRequest,
    state: &IpcState,
) -> CommandOutcome {
    match request.command {
        CommandType::ContactGet => handle_get(request, state).await,
        CommandType::ContactProfilePicture => handle_profile_picture(request, state).await,
        _ => unreachable!("contact command prefiltered"),
    }
}

async fn handle_get(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<ContactGetPayload>(
        request.id,
        request.payload,
        "contact-get payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };

    let session_manager = state.session_manager.clone();
    match session_manager.get_contact(payload).await {
        Ok(Some(contact)) => {
            session_response(request.id, Ok(contact), "failed to serialize contact")
        }
        Ok(None) => CommandOutcome::Response(crate::types::IpcResponse::failure(
            request.id,
            not_found("contact not found"),
        )),
        Err(error) => session_response(
            request.id,
            Result::<(), _>::Err(error),
            "failed to serialize contact",
        ),
    }
}

async fn handle_profile_picture(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<ContactProfilePicturePayload>(
        request.id,
        request.payload,
        "contact-profile-picture payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };

    session_response(
        request.id,
        state
            .session_manager
            .clone()
            .get_contact_profile_picture(payload)
            .await,
        "failed to serialize profile picture",
    )
}
