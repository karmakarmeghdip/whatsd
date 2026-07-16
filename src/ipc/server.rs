use std::{path::Path, sync::Arc, time::Instant};

use anyhow::{Context, Result};
use tokio::{net::UnixListener, sync::watch};
use tracing::{info, warn};

use crate::{
    config::Config,
    types::{DaemonPaths, DaemonStatus, SessionState, SessionStatus},
};

use super::{connection::handle_connection, socket};

pub(super) const IPC_PROTOCOL_VERSION: u32 = 1;

#[derive(Debug)]
pub struct IpcServer {
    state: Arc<IpcState>,
    shutdown_tx: watch::Sender<bool>,
}

#[derive(Debug)]
pub(super) struct IpcState {
    config: Config,
    started_at: Instant,
}

impl IpcServer {
    pub fn new(config: Config, started_at: Instant, shutdown_tx: watch::Sender<bool>) -> Self {
        Self {
            state: Arc::new(IpcState { config, started_at }),
            shutdown_tx,
        }
    }

    pub async fn run(self, mut shutdown_rx: watch::Receiver<bool>) -> Result<()> {
        let listener = socket::bind_socket(&self.state.config.socket_path).await?;
        info!(socket = %self.state.config.socket_path.display(), "IPC server listening");

        let accept_result = self.accept_loop(&listener, &mut shutdown_rx).await;
        socket::cleanup_socket(&self.state.config.socket_path).await;

        accept_result
    }

    async fn accept_loop(
        &self,
        listener: &UnixListener,
        shutdown_rx: &mut watch::Receiver<bool>,
    ) -> Result<()> {
        loop {
            tokio::select! {
                result = listener.accept() => {
                    let (stream, _addr) = result.context("failed to accept IPC client")?;
                    let state = Arc::clone(&self.state);
                    let shutdown_tx = self.shutdown_tx.clone();

                    tokio::spawn(async move {
                        if let Err(error) = handle_connection(stream, state, shutdown_tx).await {
                            warn!(%error, "IPC client connection failed");
                        }
                    });
                }
                changed = shutdown_rx.changed() => {
                    if changed.is_err() || *shutdown_rx.borrow() {
                        break;
                    }
                }
            }
        }

        Ok(())
    }
}

impl IpcState {
    pub(super) fn daemon_status(&self) -> DaemonStatus {
        DaemonStatus {
            version: env!("CARGO_PKG_VERSION").to_owned(),
            protocol_version: IPC_PROTOCOL_VERSION,
            uptime_seconds: self.started_at.elapsed().as_secs(),
            paths: DaemonPaths {
                socket: path_to_string(&self.config.socket_path),
                state_dir: path_to_string(&self.config.state_dir),
                database: path_to_string(&self.config.database_path),
            },
            session: SessionStatus {
                account_id: self.config.account_id.clone(),
                state: SessionState::Disconnected,
            },
        }
    }
}

fn path_to_string(path: &Path) -> String {
    path.to_string_lossy().into_owned()
}
