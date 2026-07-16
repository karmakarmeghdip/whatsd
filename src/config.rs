use std::path::PathBuf;

use anyhow::{Context, Result};
use clap::Parser;
use directories::{BaseDirs, ProjectDirs};

#[derive(Debug, Parser)]
#[command(version, about = "Linux-first WhatsApp daemon")]
pub struct Cli {
    #[arg(long, env = "WHATSD_SOCKET")]
    pub socket: Option<PathBuf>,

    #[arg(long, env = "WHATSD_STATE_DIR")]
    pub state_dir: Option<PathBuf>,

    #[arg(long, env = "WHATSD_DATABASE")]
    pub database: Option<PathBuf>,

    #[arg(long, env = "WHATSD_LOG", default_value = "info")]
    pub log_filter: String,

    #[arg(long, env = "WHATSD_ACCOUNT_ID", default_value = "default")]
    pub account_id: String,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub socket_path: PathBuf,
    pub state_dir: PathBuf,
    pub database_path: PathBuf,
    pub daemon_database_path: PathBuf,
    pub database_is_explicit: bool,
    pub log_filter: String,
    pub account_id: String,
}

impl Config {
    pub fn from_cli(cli: Cli) -> Result<Self> {
        let state_dir = match cli.state_dir {
            Some(path) => path,
            None => default_state_dir()?,
        };

        let socket_path = match cli.socket {
            Some(path) => path,
            None => default_socket_path()?,
        };

        validate_account_id(&cli.account_id)?;

        let database_is_explicit = cli.database.is_some();
        let database_path = cli
            .database
            .unwrap_or_else(|| default_database_path(&state_dir, &cli.account_id));
        let daemon_database_path = default_daemon_database_path(&state_dir, &cli.account_id);

        Ok(Self {
            socket_path,
            state_dir,
            database_path,
            daemon_database_path,
            database_is_explicit,
            log_filter: cli.log_filter,
            account_id: cli.account_id,
        })
    }
}

fn validate_account_id(account_id: &str) -> Result<()> {
    if account_id.is_empty() || matches!(account_id, "." | "..") {
        anyhow::bail!("account_id must be a safe non-empty path component");
    }

    if !account_id
        .chars()
        .all(|c| c.is_ascii_alphanumeric() || matches!(c, '_' | '-' | '.'))
    {
        anyhow::bail!("account_id may only contain ASCII letters, digits, '_', '-', and '.'");
    }

    Ok(())
}

fn default_socket_path() -> Result<PathBuf> {
    let base_dirs = BaseDirs::new().context("failed to determine base directories")?;
    let runtime_dir = base_dirs
        .runtime_dir()
        .context("XDG_RUNTIME_DIR is required unless --socket is provided")?;

    Ok(runtime_dir.join("whatsd").join("whatsd.sock"))
}

fn default_state_dir() -> Result<PathBuf> {
    let project_dirs =
        ProjectDirs::from("", "", "whatsd").context("failed to determine project directories")?;

    Ok(project_dirs
        .state_dir()
        .context("XDG_STATE_HOME could not be resolved")?
        .to_path_buf())
}

fn default_database_path(state_dir: &std::path::Path, account_id: &str) -> PathBuf {
    state_dir
        .join("accounts")
        .join(account_id)
        .join("whatsapp.db")
}

fn default_daemon_database_path(state_dir: &std::path::Path, account_id: &str) -> PathBuf {
    state_dir
        .join("accounts")
        .join(account_id)
        .join("whatsd.db")
}
