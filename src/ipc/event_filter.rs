use crate::types::EventType;

pub(super) fn supported_event_types() -> Vec<EventType> {
    vec![
        EventType::Connected,
        EventType::Disconnected,
        EventType::LoggedOut,
        EventType::PairingQr,
        EventType::PairingCode,
    ]
}

pub(super) fn is_supported_event_type(event: &EventType) -> bool {
    supported_event_types().contains(event)
}
