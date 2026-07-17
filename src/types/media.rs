use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct MediaDownloadPayload {
    pub relative_path: String,
    pub media: MediaDownloadSource,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct MediaDownloadSource {
    #[serde(rename = "type")]
    pub media_type: MediaDownloadType,
    pub direct_path: String,
    pub media_key: String,
    pub file_sha256: String,
    pub file_enc_sha256: String,
    pub file_length: u64,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum MediaDownloadType {
    Image,
    Video,
    Audio,
    Document,
    Sticker,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct MediaDownloadResultPayload {
    pub path: String,
    pub media_type: MediaDownloadType,
    pub bytes_written: u64,
}
