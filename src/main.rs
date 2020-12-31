use std::env;
use std::error;
use std::ffi::{OsStr, OsString};
use std::fs::File;
use std::io::{stdin, BufReader, Read};
use std::path::PathBuf;
use std::time::Duration;

use reqwest;
use serde::{Deserialize, Serialize};
use structopt::StructOpt;

use multifon::{Client, Routing, API};

const ENV_LOGIN: &'static str = "MULTIFON_LOGIN";
const ENV_PASSWORD: &'static str = "MULTIFON_PASSWORD";
const ENV_API: &'static str = "MULTIFON_API";
const ENV_TIMEOUT: &'static str = "MULTIFON_TIMEOUT";
const ENV_NEW_PASSWORD: &'static str = "MULTIFON_NEW_PASSWORD";
const OK: &'static str = "OK";

static mut BUG_FIX: bool = false;

fn config_parser(s: impl AsRef<OsStr>) -> Result<Config, OsString> {
    // This is ugly bug fix. For unknown reason this parser runs twice.
    // That's why we lose stdin.
    unsafe {
        if !BUG_FIX {
            BUG_FIX = true;
            return Ok(Config::default());
        }
    }
    let s = s.as_ref();
    let file: Box<dyn Read> = match s != "-" {
        true => Box::new(File::open(s).map_err(|err| OsString::from(err.to_string()))?),
        false => Box::new(stdin()),
    };
    let mut config: Config = serde_json::from_reader(BufReader::new(file))
        .map_err(|err| OsString::from(err.to_string()))?;
    config.path = s.into();
    Ok(config)
}

#[derive(StructOpt)]
struct Opts {
    ///
    #[structopt(short, long, parse(try_from_os_str = config_parser))]
    config: Option<Config>,

    /// (env: MULTIFON_LOGIN)
    #[structopt(short, long)]
    login: Option<String>,

    /// (env: MULTIFON_PASSWORD)
    #[structopt(short, long)]
    password: Option<String>,

    /// (env: MULTIFON_API)
    #[structopt(short, long)]
    api: Option<API>,

    /// (env: MULTIFON_TIMEOUT)
    #[structopt(short, long)]
    timeout: Option<u64>,

    ///
    #[structopt(subcommand)]
    command: Command,
}

#[derive(StructOpt)]
enum Command {
    ///
    Balance,

    /// get / set
    Routing { routing: Option<Routing> },

    ///
    Status,

    ///
    Profile,

    /// get / set
    Lines { lines: Option<i32> },

    /// (env: MULTIFON_NEW_PASSWORD)
    SetPassword { password: Option<String> },
}

#[derive(Serialize, Deserialize, Default, Clone)]
struct Config {
    #[serde(default)]
    login: Option<String>,

    #[serde(default)]
    password: Option<String>,

    #[serde(default)]
    new_password: Option<String>,

    #[serde(default)]
    api: Option<API>,

    #[serde(default)]
    timeout: Option<u64>,

    #[serde(skip)]
    path: PathBuf,
}

fn get_opts() -> Result<Opts, Box<dyn error::Error>> {
    let mut opts = Opts::from_args();
    let config = opts.config.clone().unwrap_or_default();
    if opts.login.is_none() {
        opts.login = match config.login {
            Some(_) => config.login,
            None => Some(env::var(ENV_LOGIN)?),
        };
    }
    env::remove_var(ENV_LOGIN);
    if opts.password.is_none() {
        opts.password = match config.password {
            Some(_) => config.password,
            None => Some(env::var(ENV_PASSWORD)?),
        };
    }
    env::remove_var(ENV_PASSWORD);
    if opts.api.is_none() {
        opts.api = match env::var(ENV_API) {
            Ok(s) => Some(s.parse()?),
            Err(_) => config.api,
        }
    }
    if opts.timeout.is_none() {
        opts.timeout = match env::var(ENV_TIMEOUT) {
            Ok(s) => Some(s.parse()?),
            Err(_) => config.timeout,
        }
    }
    if let Command::SetPassword { password: None } = opts.command {
        opts.command = Command::SetPassword {
            password: match config.new_password {
                Some(_) => config.new_password,
                None => Some(env::var(ENV_NEW_PASSWORD)?),
            },
        };
    }
    env::remove_var(ENV_NEW_PASSWORD);
    Ok(opts)
}

fn update_config(opts: &Opts) -> Result<(), Box<dyn error::Error>> {
    let mut config = opts.config.clone().unwrap();
    config.new_password = opts.password.clone();
    if let Command::SetPassword { ref password } = opts.command {
        config.password = password.clone();
    }
    let file = File::create(&config.path)?;
    serde_json::to_writer_pretty(&file, &config)?;
    Ok(())
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn error::Error>> {
    let opts = get_opts()?;
    let http_client = match opts.timeout {
        Some(n) => Some(
            reqwest::ClientBuilder::new()
                .https_only(true)
                .timeout(Duration::from_secs(n))
                .build()?,
        ),
        None => None,
    };
    let mut client = Client::new(
        opts.login.as_ref().unwrap(),
        opts.password.as_ref().unwrap(),
        opts.api,
        http_client,
    )?;
    match opts.command {
        Command::Balance => {
            println!("{}", client.get_balance().await?.value())
        }
        Command::Routing { routing } => match routing {
            Some(v) => {
                client.set_routing(v).await?;
                println!("{}", OK);
            }
            None => println!("{}", client.get_routing().await?.value()),
        },
        Command::Status => {
            let status = client.get_status().await?;
            println!("{}:{}", status.value(), status.expires.unwrap_or_default());
        }
        Command::Profile => {
            println!("{}", client.get_profile().await?.value())
        }
        Command::Lines { lines } => match lines {
            Some(n) => {
                client.set_lines(n).await?;
                println!("{}", OK);
            }
            None => println!("{}", client.get_lines().await?.value()),
        },
        Command::SetPassword { ref password } => {
            client.set_password(password.as_ref().unwrap()).await?;
            if opts.config.is_some() {
                update_config(&opts)?;
            }
            println!("{}", OK);
        }
    }
    Ok(())
}
