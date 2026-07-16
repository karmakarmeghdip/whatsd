use crate::types::{
    ErrorBody, EventSubscribePayload, EventSubscriptionStatus, IpcRequest, IpcResponse,
};

use super::{
    command::CommandOutcome,
    event_filter::{is_supported_event_type, supported_event_types},
};

pub(super) fn subscribe_response(request: IpcRequest) -> CommandOutcome {
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
