use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SendTextPayload {
    pub chat_jid: String,
    pub text: String,
}
