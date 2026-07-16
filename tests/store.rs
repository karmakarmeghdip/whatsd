use std::{fs, os::unix::fs::PermissionsExt};

use anyhow::{Context, Result};
use uuid::Uuid;
use whatsd::store::{ListMessagesQuery, MessageRecordInput, ReceiptRecordInput, Store};

#[tokio::test]
async fn store_upserts_lists_gets_and_tracks_latest_receipts() -> Result<()> {
    let test_dir = make_test_dir()?;
    let store = Store::open(test_dir.join("whatsd.db")).await?;

    store
        .upsert_message(message("msg-1", 100, "older text"))
        .await?;
    store
        .upsert_message(message("msg-2", 200, "newer text"))
        .await?;
    store
        .upsert_message(message("msg-1", 100, "updated text"))
        .await?;

    store
        .upsert_receipt(ReceiptRecordInput {
            account_id: "default".to_owned(),
            chat_jid: "123@s.whatsapp.net".to_owned(),
            sender_jid: "123@s.whatsapp.net".to_owned(),
            message_ids: vec!["msg-1".to_owned()],
            timestamp_unix_seconds: 150,
            receipt_type: "delivered".to_owned(),
            is_from_me: false,
            is_group: false,
        })
        .await?;
    store
        .upsert_receipt(ReceiptRecordInput {
            account_id: "default".to_owned(),
            chat_jid: "123@s.whatsapp.net".to_owned(),
            sender_jid: "123@s.whatsapp.net".to_owned(),
            message_ids: vec!["msg-1".to_owned()],
            timestamp_unix_seconds: 140,
            receipt_type: "read".to_owned(),
            is_from_me: false,
            is_group: false,
        })
        .await?;

    let messages = store
        .list_messages(ListMessagesQuery {
            account_id: "default".to_owned(),
            chat_jid: "123@s.whatsapp.net".to_owned(),
            limit: 50,
            before_timestamp_unix_seconds: None,
        })
        .await?;

    assert_eq!(messages.len(), 2);
    assert_eq!(messages[0].message.message_id, "msg-2");
    assert_eq!(messages[1].message.message_id, "msg-1");
    assert_eq!(messages[1].message.text.as_deref(), Some("updated text"));
    assert_eq!(
        messages[1]
            .latest_receipt
            .as_ref()
            .map(|receipt| receipt.receipt_type.as_str()),
        Some("delivered")
    );

    let message = store
        .get_message(
            "default".to_owned(),
            "123@s.whatsapp.net".to_owned(),
            "msg-1".to_owned(),
        )
        .await?
        .context("message should exist")?;
    assert_eq!(message.message.text.as_deref(), Some("updated text"));

    let missing = store
        .get_message(
            "default".to_owned(),
            "123@s.whatsapp.net".to_owned(),
            "missing".to_owned(),
        )
        .await?;
    assert!(missing.is_none());

    fs::remove_dir_all(test_dir).context("failed to remove test directory")?;
    Ok(())
}

fn message(message_id: &str, timestamp: i64, text: &str) -> MessageRecordInput {
    MessageRecordInput {
        account_id: "default".to_owned(),
        chat_jid: "123@s.whatsapp.net".to_owned(),
        sender_jid: "123@s.whatsapp.net".to_owned(),
        message_id: message_id.to_owned(),
        server_id: 0,
        timestamp_unix_seconds: timestamp,
        message_type: "text".to_owned(),
        text: Some(text.to_owned()),
        is_from_me: false,
        is_group: false,
        direction: "inbound".to_owned(),
    }
}

fn make_test_dir() -> Result<std::path::PathBuf> {
    let path = std::env::temp_dir().join(format!("whatsd-store-test-{}", Uuid::new_v4()));
    fs::create_dir(&path).context("failed to create test directory")?;
    fs::set_permissions(&path, fs::Permissions::from_mode(0o700))
        .context("failed to set test directory permissions")?;
    Ok(path)
}
