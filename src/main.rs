use anyhow::{Context, Result};
use clap::Parser;
use tracing_subscriber::EnvFilter;
use whatsd::{config::Cli, daemon};

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    init_tracing(&cli.log_filter)?;
    let config = whatsd::config::Config::from_cli(cli)?;

    daemon::run(config).await
}

fn init_tracing(log_filter: &str) -> Result<()> {
    let env_filter = EnvFilter::try_new(log_filter).context("invalid log filter")?;

    tracing_subscriber::fmt()
        .with_env_filter(env_filter)
        .try_init()
        .map_err(|error| anyhow::anyhow!("failed to initialize tracing subscriber: {error}"))
}
