pub mod client;
pub mod response;

pub use client::{Client, Error, Result, Routing, API, DEFAULT_TIMEOUT};

#[cfg(test)]
mod tests {
    use std::fs::File;
    use std::io::BufReader;
    use std::sync::Once;
    use std::thread::sleep;
    use std::time::Duration;

    use serde::Deserialize;

    use super::*;

    static ONCE: Once = Once::new();
    static mut CONFIG: Option<Config> = None;

    #[derive(Deserialize, Clone)]
    struct Config {
        login: String,
        password: String,
        #[serde(default)]
        new_password: Option<String>,
        #[serde(default)]
        api: Option<API>,
    }

    fn delay() {
        sleep(Duration::from_secs(1));
    }

    fn client() -> Client {
        ONCE.call_once(|| {
            let reader = BufReader::new(File::open("./testdata/multifon.json").unwrap());
            let config: Option<Config> = Some(serde_json::from_reader(reader).unwrap());
            unsafe {
                CONFIG = config;
            }
        });
        delay();
        let login;
        let password;
        let api;
        unsafe {
            let conf = CONFIG.clone().unwrap();
            login = conf.login;
            password = conf.password;
            api = conf.api;
        }
        Client::new(login, password, api, None).unwrap()
    }

    #[tokio::test]
    async fn get_balance() -> Result<()> {
        client().get_balance().await.map(|_| ())
    }

    #[tokio::test]
    async fn get_routing() -> Result<()> {
        client().get_routing().await.map(|_| ())
    }

    #[tokio::test]
    async fn set_routing() -> Result<()> {
        let client = client();
        let routing = client.get_routing().await?;
        for val in &[Routing::Gsm, Routing::Sip, Routing::SipGsm] {
            if *val == routing.routing {
                continue;
            }
            delay();
            client.set_routing(val).await?;
        }
        delay();
        client.set_routing(routing.routing).await
    }

    #[tokio::test]
    async fn get_status() -> Result<()> {
        client().get_status().await.map(|_| ())
    }

    #[tokio::test]
    async fn get_profile() -> Result<()> {
        client().get_profile().await.map(|_| ())
    }

    #[tokio::test]
    async fn get_lines() -> Result<()> {
        client().get_lines().await.map(|_| ())
    }

    #[tokio::test]
    async fn set_lines() -> Result<()> {
        let client = client();
        let lines = client.get_lines().await?;
        for n in 2..4 {
            if n == lines.lines {
                continue;
            }
            delay();
            client.set_lines(n).await?;
        }
        delay();
        client.set_lines(lines.lines).await
    }

    #[tokio::test]
    #[ignore]
    async fn set_password() -> Result<()> {
        let mut client = client();
        let password;
        let new_password;
        unsafe {
            let conf = CONFIG.clone().unwrap();
            password = conf.password;
            new_password = conf.new_password;
        }
        let new_password = new_password.unwrap();
        if new_password.is_empty() {
            panic!("new_password is empty");
        }
        for passwd in &[new_password, password] {
            client.set_password(passwd).await?;
            delay();
        }
        Ok(())
    }
}
