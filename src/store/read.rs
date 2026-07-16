use anyhow::Result;
use diesel::{
    RunQueryDsl, SqliteConnection, sql_query,
    sql_types::{BigInt, Text},
};

use crate::store::{
    ListMessagesQuery, MessageRecordWithReceipt,
    row::{MessageRow, row_to_record},
};

pub(super) fn list_messages_sync(
    conn: &mut SqliteConnection,
    query: ListMessagesQuery,
) -> Result<Vec<MessageRecordWithReceipt>> {
    let before = query.before_timestamp_unix_seconds.unwrap_or(i64::MAX);
    let rows = sql_query(message_select_sql(
        "m.account_id = ? AND m.chat_jid = ? AND m.timestamp_unix_seconds < ?
         ORDER BY m.timestamp_unix_seconds DESC, m.message_id DESC LIMIT ?",
    ))
    .bind::<Text, _>(&query.account_id)
    .bind::<Text, _>(&query.chat_jid)
    .bind::<BigInt, _>(before)
    .bind::<BigInt, _>(query.limit)
    .load::<MessageRow>(conn)?;

    Ok(rows.into_iter().map(row_to_record).collect())
}

pub(super) fn get_message_sync(
    conn: &mut SqliteConnection,
    account_id: String,
    chat_jid: String,
    message_id: String,
) -> Result<Option<MessageRecordWithReceipt>> {
    let mut rows = sql_query(message_select_sql(
        "m.account_id = ? AND m.chat_jid = ? AND m.message_id = ?
         ORDER BY m.timestamp_unix_seconds DESC LIMIT 1",
    ))
    .bind::<Text, _>(&account_id)
    .bind::<Text, _>(&chat_jid)
    .bind::<Text, _>(&message_id)
    .load::<MessageRow>(conn)?;

    Ok(rows.pop().map(row_to_record))
}

fn message_select_sql(where_clause: &str) -> String {
    format!(
        "SELECT
            m.account_id, m.chat_jid, m.sender_jid, m.message_id, m.server_id,
            m.timestamp_unix_seconds, m.message_type, m.text, m.is_from_me,
            m.is_group, m.direction, r.receipt_type, r.receipt_timestamp_unix_seconds,
            r.receipt_sender_jid
        FROM messages m
        LEFT JOIN message_receipts r
            ON r.account_id = m.account_id
            AND r.chat_jid = m.chat_jid
            AND r.message_id = m.message_id
        WHERE {where_clause}"
    )
}
