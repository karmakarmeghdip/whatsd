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
    let config = test_config(&test_dir);
    let socket_path = config.socket_path.clone();

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

#[tokio::test]
async fn session_status_and_event_subscription_over_unix_socket() -> Result<()> {
    let test_dir = make_test_dir()?;
    let config = test_config(&test_dir);
    let socket_path = config.socket_path.clone();
    let expected_database = config.database_path.to_string_lossy().into_owned();
    let expected_daemon_database = config.daemon_database_path.to_string_lossy().into_owned();

    let daemon_task = tokio::spawn(async move { daemon::run(config).await });
    let stream = connect_with_retry(&socket_path).await?;
    let mut reader = BufReader::new(stream);

    let status_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": status_id,
            "type": "session.status",
            "payload": {},
        }),
    )
    .await?;
    assert_eq!(response["id"], status_id.to_string());
    assert_eq!(response["ok"], true);
    assert_eq!(response["payload"]["account_id"], "default");
    assert_eq!(response["payload"]["state"], "disconnected");

    let daemon_status_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": daemon_status_id,
            "type": "daemon.status",
            "payload": {},
        }),
    )
    .await?;
    assert_eq!(response["id"], daemon_status_id.to_string());
    assert_eq!(response["ok"], true);
    assert_eq!(response["payload"]["paths"]["database"], expected_database);
    assert_eq!(
        response["payload"]["paths"]["daemon_database"],
        expected_daemon_database
    );
    assert_eq!(response["payload"]["session"]["state"], "disconnected");

    let subscribe_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": subscribe_id,
            "type": "event.subscribe",
            "payload": { "types": ["event.connected"] },
        }),
    )
    .await?;
    assert_eq!(response["id"], subscribe_id.to_string());
    assert_eq!(response["ok"], true);
    assert_eq!(response["payload"]["subscribed"], true);
    assert_eq!(response["payload"]["types"][0], "event.connected");

    let subscribe_all_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": subscribe_all_id,
            "type": "event.subscribe",
        }),
    )
    .await?;
    assert_eq!(response["id"], subscribe_all_id.to_string());
    assert_eq!(response["ok"], true);
    assert_eq!(response["payload"]["subscribed"], true);
    assert_eq!(
        response["payload"]["types"].as_array().map(Vec::len),
        Some(8)
    );
    assert!(
        response["payload"]["types"]
            .as_array()
            .is_some_and(|types| {
                types.contains(&serde_json::Value::String("event.message".to_owned()))
                    && types.contains(&serde_json::Value::String("event.receipt".to_owned()))
            })
    );

    let pair_code_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": pair_code_id,
            "type": "session.pair_code",
            "payload": { "phone_number": "" },
        }),
    )
    .await?;
    assert_eq!(response["id"], pair_code_id.to_string());
    assert_eq!(response["ok"], false);
    assert_eq!(response["error"]["code"], "invalid_request");

    let send_text_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": send_text_id,
            "type": "message.send_text",
            "payload": {
                "chat_jid": "123456789@s.whatsapp.net",
                "text": "hello"
            },
        }),
    )
    .await?;
    assert_eq!(response["id"], send_text_id.to_string());
    assert_eq!(response["ok"], false);
    assert_eq!(response["error"]["code"], "invalid_state");

    let send_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": send_id,
            "type": "message.send",
            "payload": {
                "chat_jid": "123456789@s.whatsapp.net",
                "message": {
                    "type": "text",
                    "text": "hello"
                }
            },
        }),
    )
    .await?;
    assert_eq!(response["id"], send_id.to_string());
    assert_eq!(response["ok"], false);
    assert_eq!(response["error"]["code"], "invalid_state");

    let bare_jid_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": bare_jid_id,
            "type": "message.send_text",
            "payload": {
                "chat_jid": "123456789",
                "text": "hello"
            },
        }),
    )
    .await?;
    assert_eq!(response["id"], bare_jid_id.to_string());
    assert_eq!(response["ok"], false);
    assert_eq!(response["error"]["code"], "invalid_request");

    let list_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": list_id,
            "type": "message.list",
            "payload": {
                "chat_jid": "123456789@s.whatsapp.net",
                "limit": 10
            },
        }),
    )
    .await?;
    assert_eq!(response["id"], list_id.to_string());
    assert_eq!(response["ok"], true);
    assert_eq!(
        response["payload"]["messages"].as_array().map(Vec::len),
        Some(0)
    );

    let get_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": get_id,
            "type": "message.get",
            "payload": {
                "chat_jid": "123456789@s.whatsapp.net",
                "message_id": "missing"
            },
        }),
    )
    .await?;
    assert_eq!(response["id"], get_id.to_string());
    assert_eq!(response["ok"], false);
    assert_eq!(response["error"]["code"], "not_found");

    assert_invalid_state(
        &mut reader,
        "message.react",
        json!({
            "chat_jid": "123456789@s.whatsapp.net",
            "message_id": "msg-1",
            "emoji": "👍"
        }),
    )
    .await?;
    assert_invalid_state(
        &mut reader,
        "message.edit",
        json!({
            "chat_jid": "123456789@s.whatsapp.net",
            "message_id": "msg-1",
            "text": "edited"
        }),
    )
    .await?;
    assert_invalid_state(
        &mut reader,
        "message.revoke",
        json!({
            "chat_jid": "123456789@s.whatsapp.net",
            "message_id": "msg-1"
        }),
    )
    .await?;
    assert_invalid_state(
        &mut reader,
        "message.mark_read",
        json!({
            "chat_jid": "123456789@s.whatsapp.net",
            "message_ids": ["msg-1"],
            "sender_jid": "123456789@s.whatsapp.net"
        }),
    )
    .await?;
    assert_invalid_state(
        &mut reader,
        "presence.set",
        json!({ "status": "available" }),
    )
    .await?;
    assert_invalid_state(
        &mut reader,
        "presence.subscribe",
        json!({ "jid": "123456789@s.whatsapp.net" }),
    )
    .await?;
    assert_invalid_state(
        &mut reader,
        "presence.unsubscribe",
        json!({ "jid": "123456789@s.whatsapp.net" }),
    )
    .await?;
    assert_invalid_state(
        &mut reader,
        "chatstate.send",
        json!({
            "chat_jid": "123456789@s.whatsapp.net",
            "state": "composing"
        }),
    )
    .await?;

    let unsubscribe_id = Uuid::new_v4();
    let response = request_response(
        &mut reader,
        json!({
            "id": unsubscribe_id,
            "type": "event.unsubscribe",
            "payload": {},
        }),
    )
    .await?;
    assert_eq!(response["id"], unsubscribe_id.to_string());
    assert_eq!(response["ok"], true);
    assert_eq!(response["payload"]["subscribed"], false);

    shutdown_daemon(&mut reader).await?;
    daemon_task.await.context("daemon task failed")??;
    fs::remove_dir_all(test_dir).context("failed to remove test directory")?;

    Ok(())
}

fn test_config(test_dir: &Path) -> Config {
    let state_dir = test_dir.join("state");
    let database_path = state_dir
        .join("accounts")
        .join("default")
        .join("whatsapp.db");
    let daemon_database_path = state_dir.join("accounts").join("default").join("whatsd.db");

    Config {
        socket_path: test_dir.join("run").join("whatsd.sock"),
        state_dir,
        database_path,
        daemon_database_path,
        database_is_explicit: false,
        log_filter: "off".to_owned(),
        account_id: "default".to_owned(),
    }
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

async fn request_response(
    reader: &mut BufReader<UnixStream>,
    request: serde_json::Value,
) -> Result<serde_json::Value> {
    write_json_line(reader.get_mut(), request).await?;

    let mut response_line = String::new();
    reader
        .read_line(&mut response_line)
        .await
        .context("failed to read response")?;

    serde_json::from_str(&response_line).context("failed to parse response")
}

async fn shutdown_daemon(reader: &mut BufReader<UnixStream>) -> Result<()> {
    let shutdown_id = Uuid::new_v4();
    let response = request_response(
        reader,
        json!({
            "id": shutdown_id,
            "type": "daemon.shutdown",
            "payload": {},
        }),
    )
    .await?;

    assert_eq!(response["id"], shutdown_id.to_string());
    assert_eq!(response["ok"], true);
    Ok(())
}

async fn assert_invalid_state(
    reader: &mut BufReader<UnixStream>,
    command: &str,
    payload: serde_json::Value,
) -> Result<()> {
    let request_id = Uuid::new_v4();
    let response = request_response(
        reader,
        json!({
            "id": request_id,
            "type": command,
            "payload": payload,
        }),
    )
    .await?;

    assert_eq!(response["id"], request_id.to_string());
    assert_eq!(response["ok"], false);
    assert_eq!(response["error"]["code"], "invalid_state");
    Ok(())
}
