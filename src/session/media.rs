use std::{
    io::ErrorKind,
    os::unix::fs::PermissionsExt,
    path::{Component, Path, PathBuf},
};

use base64::{Engine, engine::general_purpose};
use tokio::fs;
use uuid::Uuid;
use whatsapp_rust::download::MediaType;

use crate::types::{
    MediaDownloadPayload, MediaDownloadResultPayload, MediaDownloadSource, MediaDownloadType,
};

use super::{error::SessionError, manager::SessionManager};

const MEDIA_CACHE_DIR: &str = "media";
const HASH_LENGTH: usize = 32;

struct DecodedMediaSource {
    media_type: MediaDownloadType,
    upstream_media_type: MediaType,
    direct_path: String,
    media_key: Vec<u8>,
    file_sha256: Vec<u8>,
    file_enc_sha256: Vec<u8>,
    file_length: u64,
}

impl SessionManager {
    pub async fn download_media(
        &self,
        payload: MediaDownloadPayload,
    ) -> Result<MediaDownloadResultPayload, SessionError> {
        let relative_path = safe_relative_path(&payload.relative_path)?;
        let source = decode_media_source(payload.media)?;
        let client = self.client_for_connected().await?;
        let media_root = self.media_cache_root();
        crate::unix::ensure_private_dir(&media_root, "media cache directory")
            .await
            .map_err(SessionError::MediaFailed)?;

        let output_path = media_root.join(relative_path);
        let parent = output_path.parent().ok_or_else(|| {
            SessionError::InvalidRequest("relative_path must include a file name".to_owned())
        })?;
        ensure_parent_dirs(&media_root, parent).await?;
        ensure_destination_available(&output_path).await?;

        let tmp_path = temporary_path(parent, &output_path)?;
        let file = std::fs::OpenOptions::new()
            .read(true)
            .write(true)
            .create_new(true)
            .open(&tmp_path)
            .map_err(|error| SessionError::MediaFailed(error.into()))?;

        let download_result = client
            .download_from_params_to_writer(
                &source.direct_path,
                &source.media_key,
                &source.file_sha256,
                &source.file_enc_sha256,
                source.file_length,
                source.upstream_media_type,
                file,
            )
            .await;

        if let Err(error) = download_result {
            cleanup_temp_file(&tmp_path).await;
            return Err(SessionError::MediaFailed(error));
        }

        let bytes_written = fs::metadata(&tmp_path)
            .await
            .map_err(|error| SessionError::MediaFailed(error.into()))?
            .len();

        if let Err(error) = fs::hard_link(&tmp_path, &output_path).await {
            cleanup_temp_file(&tmp_path).await;
            return if error.kind() == ErrorKind::AlreadyExists {
                Err(SessionError::InvalidRequest(
                    "relative_path destination already exists".to_owned(),
                ))
            } else {
                Err(SessionError::MediaFailed(error.into()))
            };
        }
        cleanup_temp_file(&tmp_path).await;

        Ok(MediaDownloadResultPayload {
            path: output_path.to_string_lossy().into_owned(),
            media_type: source.media_type,
            bytes_written,
        })
    }

    fn media_cache_root(&self) -> PathBuf {
        self.cache_dir
            .join("accounts")
            .join(&self.account_id)
            .join(MEDIA_CACHE_DIR)
    }
}

fn decode_media_source(source: MediaDownloadSource) -> Result<DecodedMediaSource, SessionError> {
    if source.direct_path.trim().is_empty() {
        return Err(SessionError::InvalidRequest(
            "media.direct_path must be non-empty".to_owned(),
        ));
    }

    if source.file_length == 0 {
        return Err(SessionError::InvalidRequest(
            "media.file_length must be greater than zero".to_owned(),
        ));
    }

    Ok(DecodedMediaSource {
        media_type: source.media_type,
        upstream_media_type: upstream_media_type(source.media_type),
        direct_path: source.direct_path,
        media_key: decode_hash(&source.media_key, "media.media_key")?,
        file_sha256: decode_hash(&source.file_sha256, "media.file_sha256")?,
        file_enc_sha256: decode_hash(&source.file_enc_sha256, "media.file_enc_sha256")?,
        file_length: source.file_length,
    })
}

fn upstream_media_type(media_type: MediaDownloadType) -> MediaType {
    match media_type {
        MediaDownloadType::Image => MediaType::Image,
        MediaDownloadType::Video => MediaType::Video,
        MediaDownloadType::Audio => MediaType::Audio,
        MediaDownloadType::Document => MediaType::Document,
        MediaDownloadType::Sticker => MediaType::Sticker,
    }
}

fn decode_hash(value: &str, field_name: &str) -> Result<Vec<u8>, SessionError> {
    let decoded = general_purpose::STANDARD
        .decode(value)
        .or_else(|_| general_purpose::URL_SAFE.decode(value))
        .or_else(|_| general_purpose::URL_SAFE_NO_PAD.decode(value))
        .map_err(|_error| {
            SessionError::InvalidRequest(format!("{field_name} must be valid base64"))
        })?;

    if decoded.len() != HASH_LENGTH {
        return Err(SessionError::InvalidRequest(format!(
            "{field_name} must decode to {HASH_LENGTH} bytes"
        )));
    }

    Ok(decoded)
}

fn safe_relative_path(value: &str) -> Result<PathBuf, SessionError> {
    if value.trim().is_empty() {
        return Err(invalid_relative_path());
    }

    let path = Path::new(value);
    if path.is_absolute() {
        return Err(invalid_relative_path());
    }

    let mut clean = PathBuf::new();
    for component in path.components() {
        match component {
            Component::Normal(part) => clean.push(part),
            _ => return Err(invalid_relative_path()),
        }
    }

    if clean.as_os_str().is_empty() {
        return Err(invalid_relative_path());
    }

    Ok(clean)
}

fn invalid_relative_path() -> SessionError {
    SessionError::InvalidRequest(
        "relative_path must be a relative cache path without '.' or '..' components".to_owned(),
    )
}

async fn ensure_destination_available(output_path: &Path) -> Result<(), SessionError> {
    match fs::symlink_metadata(output_path).await {
        Ok(_) => Err(SessionError::InvalidRequest(
            "relative_path destination already exists".to_owned(),
        )),
        Err(error) if error.kind() == ErrorKind::NotFound => Ok(()),
        Err(error) => Err(SessionError::MediaFailed(error.into())),
    }
}

async fn ensure_parent_dirs(media_root: &Path, parent: &Path) -> Result<(), SessionError> {
    let relative_parent = parent.strip_prefix(media_root).map_err(|_error| {
        SessionError::InvalidRequest(
            "relative_path must stay within the media cache directory".to_owned(),
        )
    })?;

    let mut current = media_root.to_path_buf();
    for component in relative_parent.components() {
        match component {
            Component::Normal(part) => current.push(part),
            _ => {
                return Err(SessionError::InvalidRequest(
                    "relative_path must stay within the media cache directory".to_owned(),
                ));
            }
        }

        match fs::symlink_metadata(&current).await {
            Ok(metadata) if metadata.file_type().is_dir() => {}
            Ok(_) => {
                return Err(SessionError::InvalidRequest(format!(
                    "relative_path parent is not a directory: {}",
                    current.display()
                )));
            }
            Err(error) if error.kind() == ErrorKind::NotFound => {
                fs::create_dir(&current)
                    .await
                    .map_err(|error| SessionError::MediaFailed(error.into()))?;
                fs::set_permissions(&current, std::fs::Permissions::from_mode(0o700))
                    .await
                    .map_err(|error| SessionError::MediaFailed(error.into()))?;
            }
            Err(error) => return Err(SessionError::MediaFailed(error.into())),
        }
    }

    Ok(())
}

fn temporary_path(parent: &Path, output_path: &Path) -> Result<PathBuf, SessionError> {
    let file_name = output_path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or_else(|| {
            SessionError::InvalidRequest("relative_path must include a file name".to_owned())
        })?;

    Ok(parent.join(format!(".{file_name}.part-{}", Uuid::new_v4())))
}

async fn cleanup_temp_file(path: &Path) {
    if let Err(error) = fs::remove_file(path).await
        && error.kind() != ErrorKind::NotFound
    {
        tracing::warn!(path = %path.display(), %error, "failed to remove media temp file");
    }
}
