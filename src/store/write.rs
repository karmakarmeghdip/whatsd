use anyhow::Result;
use diesel::{
    RunQueryDsl, SqliteConnection, sql_query,
    sql_types::{BigInt, Bool, Integer, Nullable, Text},
};

use crate::store::{MessageRecordInput, ReceiptRecordInput, connection::now_unix_seconds};

pub(super) fn upsert_message_sync(
    conn: &mut SqliteConnection,
    message: MessageRecordInput,
) -> Result<()> {
    let now = now_unix_seconds();
    sql_query(
        "INSERT INTO messages (
            account_id, chat_jid, sender_jid, message_id, server_id,
            timestamp_unix_seconds, message_type, text, is_from_me, is_group,
            direction, created_at_unix_seconds, updated_at_unix_seconds
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(account_id, chat_jid, sender_jid, message_id) DO UPDATE SET
            server_id = excluded.server_id,
            timestamp_unix_seconds = excluded.timestamp_unix_seconds,
            message_type = excluded.message_type,
            text = excluded.text,
            is_from_me = excluded.is_from_me,
            is_group = excluded.is_group,
            direction = excluded.direction,
            updated_at_unix_seconds = excluded.updated_at_unix_seconds",
    )
    .bind::<Text, _>(&message.account_id)
    .bind::<Text, _>(&message.chat_jid)
    .bind::<Text, _>(&message.sender_jid)
    .bind::<Text, _>(&message.message_id)
    .bind::<Integer, _>(message.server_id)
    .bind::<BigInt, _>(message.timestamp_unix_seconds)
    .bind::<Text, _>(&message.message_type)
    .bind::<Nullable<Text>, _>(&message.text)
    .bind::<Bool, _>(message.is_from_me)
    .bind::<Bool, _>(message.is_group)
    .bind::<Text, _>(&message.direction)
    .bind::<BigInt, _>(now)
    .bind::<BigInt, _>(now)
    .execute(conn)?;

    upsert_cursor(
        conn,
        &message.account_id,
        "event.message",
        message.timestamp_unix_seconds,
        Some(&message.message_id),
    )
}

pub(super) fn upsert_receipt_sync(
    conn: &mut SqliteConnection,
    receipt: ReceiptRecordInput,
) -> Result<()> {
    for message_id in &receipt.message_ids {
        upsert_single_receipt(conn, &receipt, message_id)?;
    }

    upsert_cursor(
        conn,
        &receipt.account_id,
        "event.receipt",
        receipt.timestamp_unix_seconds,
        receipt.message_ids.first().map(String::as_str),
    )
}

fn upsert_single_receipt(
    conn: &mut SqliteConnection,
    receipt: &ReceiptRecordInput,
    message_id: &str,
) -> Result<()> {
    sql_query(
        "INSERT INTO message_receipts (
            account_id, chat_jid, message_id, receipt_type,
            receipt_timestamp_unix_seconds, receipt_sender_jid, is_from_me,
            is_group, updated_at_unix_seconds
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(account_id, chat_jid, message_id) DO UPDATE SET
            receipt_type = excluded.receipt_type,
            receipt_timestamp_unix_seconds = excluded.receipt_timestamp_unix_seconds,
            receipt_sender_jid = excluded.receipt_sender_jid,
            is_from_me = excluded.is_from_me,
            is_group = excluded.is_group,
            updated_at_unix_seconds = excluded.updated_at_unix_seconds
        WHERE excluded.receipt_timestamp_unix_seconds >= message_receipts.receipt_timestamp_unix_seconds",
    )
    .bind::<Text, _>(&receipt.account_id)
    .bind::<Text, _>(&receipt.chat_jid)
    .bind::<Text, _>(message_id)
    .bind::<Text, _>(&receipt.receipt_type)
    .bind::<BigInt, _>(receipt.timestamp_unix_seconds)
    .bind::<Text, _>(&receipt.sender_jid)
    .bind::<Bool, _>(receipt.is_from_me)
    .bind::<Bool, _>(receipt.is_group)
    .bind::<BigInt, _>(now_unix_seconds())
    .execute(conn)?;
    Ok(())
}

fn upsert_cursor(
    conn: &mut SqliteConnection,
    account_id: &str,
    event_type: &str,
    timestamp_unix_seconds: i64,
    message_id: Option<&str>,
) -> Result<()> {
    sql_query(
        "INSERT INTO event_cursors (
            account_id, event_type, cursor_timestamp_unix_seconds,
            cursor_message_id, updated_at_unix_seconds
        ) VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(account_id, event_type) DO UPDATE SET
            cursor_timestamp_unix_seconds = excluded.cursor_timestamp_unix_seconds,
            cursor_message_id = excluded.cursor_message_id,
            updated_at_unix_seconds = excluded.updated_at_unix_seconds
        WHERE excluded.cursor_timestamp_unix_seconds >= event_cursors.cursor_timestamp_unix_seconds",
    )
    .bind::<Text, _>(account_id)
    .bind::<Text, _>(event_type)
    .bind::<BigInt, _>(timestamp_unix_seconds)
    .bind::<Nullable<Text>, _>(message_id)
    .bind::<BigInt, _>(now_unix_seconds())
    .execute(conn)?;
    Ok(())
}
