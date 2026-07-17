use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ContactGetPayload {
    pub jid: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ContactPayload {
    pub jid: String,
    pub lid_jid: Option<String>,
    pub status: Option<String>,
    pub picture_id: Option<String>,
    pub is_business: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ContactProfilePicturePayload {
    pub jid: String,
    #[serde(default = "default_profile_picture_preview")]
    pub preview: bool,
}

fn default_profile_picture_preview() -> bool {
    true
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ContactProfilePictureResultPayload {
    pub jid: String,
    pub picture: Option<ContactProfilePicture>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ContactProfilePicture {
    pub id: String,
    pub url: String,
    pub direct_path: Option<String>,
    pub hash: Option<String>,
}
