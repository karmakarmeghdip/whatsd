use std::{fs, os::unix::fs::PermissionsExt, path::Path, time::Duration};

use anyhow::{Context, Result};
use serde_json::json;
use tokio::{
    io::{AsyncBufReadExt, AsyncWriteExt, BufReader},
    net::UnixStream,
    time::sleep,
};
use uuid::Uuid;
use whatsd::{config::Config, daemon};

#[tokio::test]
async fn daemon_ping_over_unix_socket() -> Result<()> {
    let test_dir = make_test_dir()?;
    let socket_path = test_dir.join("run").join("whatsd.sock");
    let state_dir = test_dir.join("state");
    let database_path = state_dir.join("whatsapp.db");

    let config = Config {
        socket_path: socket_path.clone(),
        state_dir,
        database_path,
        log_filter: "off".to_owned(),
        account_id: "default".to_owned(),
    };

    let daemon_task = tokio::spawn(async move { daemon::run(config).await });
    let stream = connect_with_retry(&socket_path).await?;
    let mut reader = BufReader::new(stream);

    let request_id = Uuid::new_v4();
    write_json_line(
        reader.get_mut(),
        json!({
            "id": request_id,
            "type": "daemon.ping",
            "payload": {},
        }),
    )
        .await?;

    let mut response_line = String::new();
    reader
        .read_line(&mut response_line)
        .await
        .context("failed to read ping response")?;
    let response: serde_json::Value = serde_json::from_str(&response_line)?;

    assert_eq!(response["id"], request_id.to_string());
    assert_eq!(response["ok"], true);
    assert!(response["payload"]["timestamp_unix_seconds"].is_u64());

    let shutdown_id = Uuid::new_v4();
    write_json_line(
        reader.get_mut(),
        json!({
            "id": shutdown_id,
            "type": "daemon.shutdown",
            "payload": {},
        }),
    )
        .await?;

    response_line.clear();
    reader
        .read_line(&mut response_line)
        .await
        .context("failed to read shutdown response")?;
    let response: serde_json::Value = serde_json::from_str(&response_line)?;
    assert_eq!(response["id"], shutdown_id.to_string());
    assert_eq!(response["ok"], true);

    daemon_task.await.context("daemon task failed")??;
    fs::remove_dir_all(test_dir).context("failed to remove test directory")?;

    Ok(())
}

fn make_test_dir() -> Result<std::path::PathBuf> {
    let path = std::env::temp_dir().join(format!("whatsd-test-{}", Uuid::new_v4()));
    fs::create_dir(&path).context("failed to create test directory")?;
    fs::set_permissions(&path, fs::Permissions::from_mode(0o700))
        .context("failed to set test directory permissions")?;
    Ok(path)
}

async fn connect_with_retry(path: &Path) -> Result<UnixStream> {
    let mut last_error = None;

    for _ in 0..100 {
        match UnixStream::connect(path).await {
            Ok(stream) => return Ok(stream),
            Err(error) => {
                last_error = Some(error);
                sleep(Duration::from_millis(10)).await;
            }
        }
    }

    Err(last_error.context("missing connection error")?.into())
}

async fn write_json_line(stream: &mut UnixStream, value: serde_json::Value) -> Result<()> {
    let mut line = serde_json::to_vec(&value).context("failed to serialize request")?;
    line.push(b'\n');
    stream
        .write_all(&line)
        .await
        .context("failed to write request")?;
    stream.flush().await.context("failed to flush request")?;
    Ok(())
}
