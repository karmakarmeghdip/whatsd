use std::{path::PathBuf, sync::Arc};

use tokio::sync::{Mutex, broadcast};
use tracing::debug;
use whatsapp_rust::{Client, bot::BotHandle, pair_code::PairCodeOptions};

use crate::{
    store::Store,
    types::{DaemonEvent, EventType, PairCodePayload, SessionState, SessionStatus},
};

use super::{error::SessionError, pair_code::pair_code_options};

#[derive(Clone)]
pub struct SessionManager {
    pub(super) account_id: String,
    pub(super) database_path: PathBuf,
    pub(super) database_is_explicit: bool,
    pub(super) cache_dir: PathBuf,
    pub(super) store: Store,
    pub(super) inner: Arc<Mutex<SessionInner>>,
    pub(super) events: broadcast::Sender<DaemonEvent>,
}

pub(super) struct SessionInner {
    pub(super) state: SessionState,
    pub(super) running: Option<RunningSession>,
    pub(super) generation: u64,
}

pub(super) struct RunningSession {
    pub(super) client: Arc<Client>,
}

impl SessionManager {
    pub fn new(
        account_id: String,
        database_path: PathBuf,
        database_is_explicit: bool,
        cache_dir: PathBuf,
        store: Store,
    ) -> Self {
        let (events, _rx) = broadcast::channel(128);

        Self {
            account_id,
            database_path,
            database_is_explicit,
            cache_dir,
            store,
            inner: Arc::new(Mutex::new(SessionInner {
                state: SessionState::Disconnected,
                running: None,
                generation: 0,
            })),
            events,
        }
    }

    pub fn subscribe(&self) -> broadcast::Receiver<DaemonEvent> {
        self.events.subscribe()
    }

    pub async fn status(&self) -> SessionStatus {
        self.status_with_state(self.inner.lock().await.state)
    }

    pub async fn connect(&self) -> Result<SessionStatus, SessionError> {
        self.start(None).await
    }

    pub async fn pair_code(&self, payload: PairCodePayload) -> Result<SessionStatus, SessionError> {
        self.start(Some(pair_code_options(payload)?)).await
    }

    pub async fn disconnect(&self) -> Result<SessionStatus, SessionError> {
        let client = self.take_running(SessionState::Disconnected).await?;
        client.disconnect().await;
        self.publish_event(EventType::Disconnected, serde_json::json!({}));
        Ok(self.status().await)
    }

    pub async fn logout(&self) -> Result<SessionStatus, SessionError> {
        let client = self.take_running(SessionState::LoggedOut).await?;
        client.logout().await.map_err(SessionError::StopFailed)?;
        self.publish_event(
            EventType::LoggedOut,
            serde_json::json!({ "requested": true }),
        );
        Ok(self.status().await)
    }

    pub async fn shutdown(&self) {
        let client = {
            let mut inner = self.inner.lock().await;
            inner.running.take().map(|running| running.client)
        };

        if let Some(client) = client {
            client.disconnect().await;
        }
    }

    async fn start(
        &self,
        pair_code_options: Option<PairCodeOptions>,
    ) -> Result<SessionStatus, SessionError> {
        let generation = {
            let mut inner = self.inner.lock().await;
            if !matches!(
                inner.state,
                SessionState::Disconnected | SessionState::LoggedOut
            ) {
                return Err(SessionError::InvalidState(format!(
                    "session is {:?}",
                    inner.state
                )));
            }

            inner.generation += 1;
            inner.state = if pair_code_options.is_some() {
                SessionState::Pairing
            } else {
                SessionState::Connecting
            };
            inner.generation
        };

        match self.build_and_run_bot(pair_code_options, generation).await {
            Ok((client, handle)) => {
                self.inner.lock().await.running = Some(RunningSession { client });
                self.spawn_handle_monitor(handle, generation);
                Ok(self.status().await)
            }
            Err(error) => {
                let mut inner = self.inner.lock().await;
                if inner.generation == generation {
                    inner.state = SessionState::Disconnected;
                    inner.running = None;
                }
                Err(SessionError::StartFailed(error))
            }
        }
    }

    pub(super) fn spawn_handle_monitor(&self, handle: BotHandle, generation: u64) {
        let inner = Arc::clone(&self.inner);

        tokio::spawn(async move {
            if let Err(error) = handle.await {
                debug!(%error, "WhatsApp bot handle closed before completion");
            }

            let mut inner = inner.lock().await;
            if inner.generation == generation {
                inner.running = None;
                if !matches!(inner.state, SessionState::LoggedOut) {
                    inner.state = SessionState::Disconnected;
                }
            }
        });
    }

    pub(super) async fn client_for_connected(&self) -> Result<Arc<Client>, SessionError> {
        let inner = self.inner.lock().await;
        if inner.state != SessionState::Connected {
            return Err(SessionError::InvalidState(format!(
                "session is {:?}",
                inner.state
            )));
        }

        inner
            .running
            .as_ref()
            .map(|running| Arc::clone(&running.client))
            .ok_or_else(|| SessionError::InvalidState("session is not running".to_owned()))
    }

    async fn take_running(&self, next_state: SessionState) -> Result<Arc<Client>, SessionError> {
        let mut inner = self.inner.lock().await;
        let Some(running) = inner.running.take() else {
            return Err(SessionError::InvalidState(
                "session is not running".to_owned(),
            ));
        };

        inner.state = next_state;
        inner.generation += 1;
        Ok(running.client)
    }

    fn status_with_state(&self, state: SessionState) -> SessionStatus {
        SessionStatus {
            account_id: self.account_id.clone(),
            state,
        }
    }

    fn publish_event(&self, event: EventType, payload: serde_json::Value) {
        let mut payload = payload;
        payload["account_id"] = serde_json::Value::String(self.account_id.clone());
        let _ = self.events.send(DaemonEvent { event, payload });
    }
}
