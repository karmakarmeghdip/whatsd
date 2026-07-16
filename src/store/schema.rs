use anyhow::Result;
use diesel::{RunQueryDsl, SqliteConnection, sql_query};

pub(super) fn migrate_sync(conn: &mut SqliteConnection) -> Result<()> {
    sql_query("PRAGMA journal_mode = WAL").execute(conn)?;
    sql_query("PRAGMA foreign_keys = ON").execute(conn)?;
    sql_query(
        "CREATE TABLE IF NOT EXISTS messages (
            account_id TEXT NOT NULL,
            chat_jid TEXT NOT NULL,
            sender_jid TEXT NOT NULL,
            message_id TEXT NOT NULL,
            server_id INTEGER NOT NULL,
            timestamp_unix_seconds INTEGER NOT NULL,
            message_type TEXT NOT NULL,
            text TEXT,
            is_from_me BOOLEAN NOT NULL,
            is_group BOOLEAN NOT NULL,
            direction TEXT NOT NULL,
            created_at_unix_seconds INTEGER NOT NULL,
            updated_at_unix_seconds INTEGER NOT NULL,
            PRIMARY KEY (account_id, chat_jid, sender_jid, message_id)
        )",
    )
    .execute(conn)?;
    sql_query(
        "CREATE INDEX IF NOT EXISTS idx_messages_chat_time
            ON messages (account_id, chat_jid, timestamp_unix_seconds DESC)",
    )
    .execute(conn)?;
    sql_query(
        "CREATE TABLE IF NOT EXISTS message_receipts (
            account_id TEXT NOT NULL,
            chat_jid TEXT NOT NULL,
            message_id TEXT NOT NULL,
            receipt_type TEXT NOT NULL,
            receipt_timestamp_unix_seconds INTEGER NOT NULL,
            receipt_sender_jid TEXT NOT NULL,
            is_from_me BOOLEAN NOT NULL,
            is_group BOOLEAN NOT NULL,
            updated_at_unix_seconds INTEGER NOT NULL,
            PRIMARY KEY (account_id, chat_jid, message_id)
        )",
    )
    .execute(conn)?;
    sql_query(
        "CREATE TABLE IF NOT EXISTS event_cursors (
            account_id TEXT NOT NULL,
            event_type TEXT NOT NULL,
            cursor_timestamp_unix_seconds INTEGER NOT NULL,
            cursor_message_id TEXT,
            updated_at_unix_seconds INTEGER NOT NULL,
            PRIMARY KEY (account_id, event_type)
        )",
    )
    .execute(conn)?;
    Ok(())
}
