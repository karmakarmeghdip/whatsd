use std::sync::Arc;

use anyhow::{Context, Result};
use whatsapp_rust::{
    Client, TokioRuntime,
    bot::{Bot, BotHandle},
    pair_code::PairCodeOptions,
    store::{SqliteStore, traits::Backend},
    transport::{TokioWebSocketTransportFactory, UreqHttpClient},
};

use crate::unix::ensure_private_dir;

use super::{events::normalize_event, manager::SessionManager};

impl SessionManager {
    pub(super) async fn build_and_run_bot(
        &self,
        pair_code_options: Option<PairCodeOptions>,
        generation: u64,
    ) -> Result<(Arc<Client>, BotHandle)> {
        self.prepare_database_dir().await?;
        let database = self.database_path.to_string_lossy().into_owned();
        let backend = Arc::new(SqliteStore::new(&database).await?) as Arc<dyn Backend>;
        let event_inner = Arc::clone(&self.inner);
        let event_tx = self.events.clone();
        let event_store = self.store.clone();
        let account_id = self.account_id.clone();

        let mut builder = Bot::builder()
            .with_backend(backend)
            .with_transport_factory(TokioWebSocketTransportFactory::new())
            .with_http_client(UreqHttpClient::new())
            .with_runtime(TokioRuntime)
            .on_event(move |event, _client| {
                let event_inner = Arc::clone(&event_inner);
                let event_tx = event_tx.clone();
                let event_store = event_store.clone();
                let account_id = account_id.clone();

                async move {
                    normalize_event(
                        event,
                        event_inner,
                        event_tx,
                        event_store,
                        account_id,
                        generation,
                    )
                    .await;
                }
            });

        if let Some(options) = pair_code_options {
            builder = builder.with_pair_code(options);
        }

        let mut bot = builder.build().await?;
        let client = bot.client();
        let handle = bot.run().await?;

        Ok((client, handle))
    }

    async fn prepare_database_dir(&self) -> Result<()> {
        if self.database_is_explicit {
            return Ok(());
        }

        let parent = self
            .database_path
            .parent()
            .context("account database path has no parent directory")?;
        ensure_private_dir(parent, "account state directory").await
    }
}
