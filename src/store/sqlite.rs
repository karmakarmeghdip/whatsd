use std::path::PathBuf;

use anyhow::{Context, Result};
use tokio::task;

use crate::{
    store::{
        connection::connect,
        models::{
            ListMessagesQuery, MessageRecordInput, MessageRecordWithReceipt, ReceiptRecordInput,
        },
        read::{get_message_sync, list_messages_sync},
        schema::migrate_sync,
        write::{upsert_message_sync, upsert_receipt_sync},
    },
    unix::ensure_private_dir,
};

#[derive(Debug, Clone)]
pub struct Store {
    path: PathBuf,
}

impl Store {
    pub async fn open(path: PathBuf) -> Result<Self> {
        if let Some(parent) = path.parent() {
            ensure_private_dir(parent, "daemon database directory").await?;
        }

        let store = Self { path };
        store.migrate().await?;
        Ok(store)
    }

    pub async fn upsert_message(&self, message: MessageRecordInput) -> Result<()> {
        let path = self.path.clone();
        task::spawn_blocking(move || {
            let mut conn = connect(&path)?;
            upsert_message_sync(&mut conn, message)
        })
        .await
        .context("message upsert task failed")?
    }

    pub async fn upsert_receipt(&self, receipt: ReceiptRecordInput) -> Result<()> {
        let path = self.path.clone();
        task::spawn_blocking(move || {
            let mut conn = connect(&path)?;
            upsert_receipt_sync(&mut conn, receipt)
        })
        .await
        .context("receipt upsert task failed")?
    }

    pub async fn list_messages(
        &self,
        query: ListMessagesQuery,
    ) -> Result<Vec<MessageRecordWithReceipt>> {
        let path = self.path.clone();
        task::spawn_blocking(move || {
            let mut conn = connect(&path)?;
            list_messages_sync(&mut conn, query)
        })
        .await
        .context("message list task failed")?
    }

    pub async fn get_message(
        &self,
        account_id: String,
        chat_jid: String,
        message_id: String,
    ) -> Result<Option<MessageRecordWithReceipt>> {
        let path = self.path.clone();
        task::spawn_blocking(move || {
            let mut conn = connect(&path)?;
            get_message_sync(&mut conn, account_id, chat_jid, message_id)
        })
        .await
        .context("message get task failed")?
    }

    async fn migrate(&self) -> Result<()> {
        let path = self.path.clone();
        task::spawn_blocking(move || {
            let mut conn = connect(&path)?;
            migrate_sync(&mut conn)
        })
        .await
        .context("store migration task failed")?
    }
}
