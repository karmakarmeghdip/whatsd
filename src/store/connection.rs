use std::path::Path;

use anyhow::{Context, Result};
use diesel::{Connection, SqliteConnection};

pub(super) fn connect(path: &Path) -> Result<SqliteConnection> {
    SqliteConnection::establish(&path.to_string_lossy())
        .with_context(|| format!("failed to open daemon database: {}", path.display()))
}

pub(super) fn now_unix_seconds() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map_or(0, |duration| duration.as_secs() as i64)
}
