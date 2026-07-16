use serde::{Deserialize, Serialize};

use super::EventType;

#[derive(Debug, Clone, Deserialize, PartialEq, Eq)]
pub struct EventSubscribePayload {
    #[serde(default)]
    pub types: Vec<EventType>,
}

#[derive(Debug, Clone, Serialize, PartialEq, Eq)]
pub struct EventSubscriptionStatus {
    pub subscribed: bool,
    pub types: Vec<EventType>,
}
