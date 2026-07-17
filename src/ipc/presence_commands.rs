use crate::types::{
    ChatStateSendPayload, CommandType, IpcRequest, PresenceSetPayload, PresenceSubscriptionPayload,
};

use super::{
    command::{CommandOutcome, parse_payload, session_response},
    server::IpcState,
};

pub(super) fn is_presence_command(command: CommandType) -> bool {
    matches!(
        command,
        CommandType::PresenceSet
            | CommandType::PresenceSubscribe
            | CommandType::PresenceUnsubscribe
            | CommandType::ChatStateSend
    )
}

pub(super) async fn handle_presence_request(
    request: IpcRequest,
    state: &IpcState,
) -> CommandOutcome {
    match request.command {
        CommandType::PresenceSet => handle_set(request, state).await,
        CommandType::PresenceSubscribe => handle_subscribe(request, state).await,
        CommandType::PresenceUnsubscribe => handle_unsubscribe(request, state).await,
        CommandType::ChatStateSend => handle_chatstate(request, state).await,
        _ => unreachable!("presence command prefiltered"),
    }
}

async fn handle_set(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<PresenceSetPayload>(
        request.id,
        request.payload,
        "presence-set payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.set_presence(payload).await,
        "failed to serialize action status",
    )
}

async fn handle_subscribe(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<PresenceSubscriptionPayload>(
        request.id,
        request.payload,
        "presence-subscribe payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.subscribe_presence(payload).await,
        "failed to serialize action status",
    )
}

async fn handle_unsubscribe(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<PresenceSubscriptionPayload>(
        request.id,
        request.payload,
        "presence-unsubscribe payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.unsubscribe_presence(payload).await,
        "failed to serialize action status",
    )
}

async fn handle_chatstate(request: IpcRequest, state: &IpcState) -> CommandOutcome {
    let payload = match parse_payload::<ChatStateSendPayload>(
        request.id,
        request.payload,
        "chatstate-send payload",
    ) {
        Ok(payload) => payload,
        Err(response) => return CommandOutcome::Response(response),
    };
    session_response(
        request.id,
        state.session_manager.send_chat_state(payload).await,
        "failed to serialize action status",
    )
}
