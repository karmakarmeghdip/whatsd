use std::{
    io::ErrorKind,
    os::unix::fs::{FileTypeExt, PermissionsExt},
    path::Path,
};

use anyhow::{Context, Result, bail};
use tokio::{fs, net::UnixListener, net::UnixStream};
use tracing::{debug, warn};

use crate::unix::{ensure_owned_by_current_user, ensure_private_dir};

pub(super) async fn bind_socket(path: &Path) -> Result<UnixListener> {
    let parent = path
        .parent()
        .context("socket path must include a parent directory")?;

    ensure_private_dir(parent, "socket directory")
        .await
        .context("failed to prepare socket directory")?;
    remove_stale_socket(path).await?;

    let listener = UnixListener::bind(path)
        .with_context(|| format!("failed to bind socket: {}", path.display()))?;
    fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))
        .await
        .with_context(|| format!("failed to set socket permissions: {}", path.display()))?;

    Ok(listener)
}

pub(super) async fn cleanup_socket(path: &Path) {
    let metadata = match fs::symlink_metadata(path).await {
        Ok(metadata) => metadata,
        Err(error) if error.kind() == ErrorKind::NotFound => return,
        Err(error) => {
            warn!(socket = %path.display(), %error, "failed to inspect socket during cleanup");
            return;
        }
    };

    if !metadata.file_type().is_socket() {
        warn!(socket = %path.display(), "refusing to remove non-socket during cleanup");
        return;
    }

    if let Err(error) = ensure_owned_by_current_user(&metadata, "socket path", path) {
        warn!(socket = %path.display(), %error, "refusing to remove socket during cleanup");
        return;
    }

    if let Err(error) = fs::remove_file(path).await {
        warn!(socket = %path.display(), %error, "failed to remove socket during cleanup");
    }
}

async fn remove_stale_socket(path: &Path) -> Result<()> {
    let metadata = match fs::symlink_metadata(path).await {
        Ok(metadata) => metadata,
        Err(error) if error.kind() == ErrorKind::NotFound => return Ok(()),
        Err(error) => {
            return Err(error)
                .with_context(|| format!("failed to inspect socket path: {}", path.display()));
        }
    };

    if !metadata.file_type().is_socket() {
        bail!("socket path exists but is not a socket: {}", path.display());
    }

    ensure_owned_by_current_user(&metadata, "socket path", path)?;

    match UnixStream::connect(path).await {
        Ok(_stream) => {
            bail!("socket already accepts connections: {}", path.display());
        }
        Err(error) if error.kind() == ErrorKind::ConnectionRefused => {
            debug!(socket = %path.display(), "removing stale socket");
            fs::remove_file(path)
                .await
                .with_context(|| format!("failed to remove stale socket: {}", path.display()))?;
        }
        Err(error) if error.kind() == ErrorKind::NotFound => {}
        Err(error) => {
            return Err(error)
                .with_context(|| format!("failed to probe existing socket: {}", path.display()));
        }
    }

    Ok(())
}
