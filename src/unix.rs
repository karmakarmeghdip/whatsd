use std::{io::ErrorKind, os::unix::fs::MetadataExt, os::unix::fs::PermissionsExt, path::Path};

use anyhow::{Context, Result, bail};
use tokio::fs;

pub(crate) async fn ensure_private_dir(path: &Path, label: &str) -> Result<()> {
    match fs::symlink_metadata(path).await {
        Ok(metadata) => {
            if !metadata.file_type().is_dir() {
                bail!("{label} is not a directory: {}", path.display());
            }

            ensure_owned_by_current_user(&metadata, label, path)?;

            let mode = metadata.permissions().mode() & 0o777;
            if mode & 0o077 != 0 {
                bail!(
                    "{label} must be user-only (0700): {} currently has mode {:03o}",
                    path.display(),
                    mode
                );
            }
        }
        Err(error) if error.kind() == ErrorKind::NotFound => {
            fs::create_dir_all(path)
                .await
                .with_context(|| format!("failed to create {label}: {}", path.display()))?;
            fs::set_permissions(path, std::fs::Permissions::from_mode(0o700))
                .await
                .with_context(|| {
                    format!("failed to set {label} permissions: {}", path.display())
                })?;
        }
        Err(error) => {
            return Err(error)
                .with_context(|| format!("failed to inspect {label}: {}", path.display()));
        }
    }

    Ok(())
}

pub(crate) fn ensure_owned_by_current_user(
    metadata: &std::fs::Metadata,
    label: &str,
    path: &Path,
) -> Result<()> {
    let current_uid = current_euid();
    if metadata.uid() != current_uid {
        bail!(
            "{label} must be owned by current user uid {current_uid}: {} is owned by uid {}",
            path.display(),
            metadata.uid()
        );
    }

    Ok(())
}

pub(crate) fn current_euid() -> u32 {
    unsafe { libc::geteuid() }
}
