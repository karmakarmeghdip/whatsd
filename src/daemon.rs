use std::time::Instant;

use anyhow::{Context, Result};
use tokio::sync::watch;
use tracing::info;

use crate::{config::Config, ipc::IpcServer, session::SessionManager, unix::ensure_private_dir};

pub async fn run(config: Config) -> Result<()> {
    ensure_private_dir(&config.state_dir, "state directory")
        .await
        .context("failed to prepare state directory")?;

    let started_at = Instant::now();
    let (shutdown_tx, shutdown_rx) = watch::channel(false);
    let session_manager = SessionManager::new(
        config.account_id.clone(),
        config.database_path.clone(),
        config.database_is_explicit,
    );

    let server = IpcServer::new(
        config,
        started_at,
        shutdown_tx.clone(),
        session_manager.clone(),
    );
    let mut server_task = tokio::spawn(async move { server.run(shutdown_rx).await });
    let mut shutdown_rx = shutdown_tx.subscribe();

    tokio::select! {
        result = &mut server_task => {
            result.context("IPC server task failed")??;
            return Ok(());
        }
        signal = shutdown_signal() => {
            let signal = signal?;
            info!(%signal, "received shutdown signal");
            let _ = shutdown_tx.send(true);
        }
        _ = wait_for_shutdown(&mut shutdown_rx) => {
            info!("received IPC shutdown request");
        }
    }

    let _ = shutdown_tx.send(true);
    session_manager.shutdown().await;
    server_task.await.context("IPC server task failed")??;

    Ok(())
}

async fn wait_for_shutdown(shutdown_rx: &mut watch::Receiver<bool>) {
    while !*shutdown_rx.borrow() {
        if shutdown_rx.changed().await.is_err() {
            break;
        }
    }
}

#[cfg(unix)]
async fn shutdown_signal() -> Result<&'static str> {
    let mut terminate = tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
        .context("failed to install SIGTERM handler")?;

    tokio::select! {
        signal = tokio::signal::ctrl_c() => {
            signal.context("failed to listen for Ctrl-C")?;
            Ok("ctrl-c")
        }
        _ = terminate.recv() => Ok("sigterm"),
    }
}

#[cfg(not(unix))]
async fn shutdown_signal() -> Result<&'static str> {
    tokio::signal::ctrl_c()
        .await
        .context("failed to listen for Ctrl-C")?;
    Ok("ctrl-c")
}
