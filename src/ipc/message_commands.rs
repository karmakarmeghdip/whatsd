use crate::types::{
    CommandType, GetMessagePayload, IpcRequest, IpcResponse, ListMessagesPayload,
    MessageEditPayload, MessageMarkReadPayload, MessageReactPayload, MessageRevokePayload,
    SendMessagePayload, SendTextPayload,
};

use super::{
    command::{CommandOutcome, not_found, parse_payload, session_response},
    server::IpcState,
};

pub(super) fn is_message_command(command: CommandType) -> bool {
    matches!(
        command,
        CommandType::MessageSendText
            | CommandType::MessageSend
            | CommandType::MessageList
            | CommandType::MessageGet
            | CommandType::MessageReact
            | CommandType::MessageEdit
            | CommandType::MessageRevoke
            | CommandType::MessageMarkRead
    )
}

pub(super) async fn handle_message_request(
    request: IpcRequest,
    state: &IpcState,
) -> CommandOutcome {
    match request.command {
        CommandType::MessageSendText => handle_send_text(request, state).await,
        CommandType::MessageSend => handle_send(request, state).await,
        CommandType::MessageList => handle_list(request, state).await,
        CommandType::MessageGet => handle_get(request, state).await,
        CommandType::MessageReact => handle_react(request, state).await,
        CommandType::MessageEdit => handle_edit(request, state).await,
        CommandType::MessageRevoke => handle_revoke(request, state).await,
        CommandType::MessageMarkRead => handle_mark_read(request, state).await,
        _ => unreachable!("message command prefiltered"),
    }
}

async fn handle_send_text(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload =
        match parse_payload::<SendTextPayload>(request.id, request.payload, "send-text payload") {
            Ok(payload) => payload,
            Err(response) => return CommandOutcome::Response(response),
        };
    session_response(
        request.id,
        state.session_manager.send_text(payload).await,
        "failed to serialize send result",
    )
}

async fn handle_send(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<SendMessagePayload>(
        request.id,
        request.payload,
        "send-message payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.send_message(payload).await,
        "failed to serialize send result",
    )
}

async fn handle_list(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<ListMessagesPayload>(
        request.id,
        request.payload,
        "message-list payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.list_messages(payload).await,
        "failed to serialize message list",
    )
}

async fn handle_get(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<GetMessagePayload>(
        request.id,
        request.payload,
        "message-get payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };

    match state.session_manager.get_message(payload).await {
        Ok(Some(message)) => {
            session_response(request.id, Ok(message), "failed to serialize message")
        }
        Ok(None) => CommandOutcome::Response(IpcResponse::failure(
            request.id,
            not_found("message not found"),
        )),
        Err(error) => session_response(
            request.id,
            Result::<(), _>::Err(error),
            "failed to serialize message",
        ),
    }
}

async fn handle_react(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<MessageReactPayload>(
        request.id,
        request.payload,
        "message-react payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.react_message(payload).await,
        "failed to serialize action status",
    )
}

async fn handle_edit(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<MessageEditPayload>(
        request.id,
        request.payload,
        "message-edit payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.edit_message(payload).await,
        "failed to serialize action status",
    )
}

async fn handle_revoke(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<MessageRevokePayload>(
        request.id,
        request.payload,
        "message-revoke payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.revoke_message(payload).await,
        "failed to serialize action status",
    )
}

async fn handle_mark_read(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<MessageMarkReadPayload>(
        request.id,
        request.payload,
        "message-mark-read payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.mark_read(payload).await,
        "failed to serialize action status",
    )
}
