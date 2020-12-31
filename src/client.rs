use std::error;
use std::fmt;
use std::result;
use std::str::FromStr;
use std::time::Duration;

use quick_xml;
use reqwest;
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};
use serde_repr::Deserialize_repr;

use crate::response;

pub type Result<T> = result::Result<T, Error>;

#[derive(Debug, Copy, Clone, PartialEq, Serialize, Deserialize)]
pub enum API {
    Multifon,
    Emotion,
}

#[derive(Deserialize_repr, Debug, Copy, Clone, PartialEq)]
#[repr(i32)]
pub enum Routing {
    Gsm = 0,
    Sip = 1,
    SipGsm = 2,
}

#[derive(Deserialize_repr, Debug, Copy, Clone, PartialEq)]
#[repr(i32)]
pub enum Status {
    Active = 0,
    Blocked = 1,
}

pub const DEFAULT_TIMEOUT: u64 = 1 << 5;

impl API {
    const MULTIFON: &'static str = "multifon";
    const EMOTION: &'static str = "emotion";

    fn value(&self) -> &'static str {
        match self {
            Self::Multifon => "https://sm.megafon.ru/sm/client",
            Self::Emotion => "https://emotion.megalabs.ru/sm/client",
        }
    }
}

impl FromStr for API {
    type Err = String;

    fn from_str(s: &str) -> result::Result<Self, Self::Err> {
        match s.to_uppercase().as_str() {
            Self::MULTIFON => Ok(Self::Multifon),
            Self::EMOTION => Ok(Self::Emotion),
            _ => Err(format!("invalid value: {}", s)),
        }
    }
}

impl fmt::Display for API {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let s = match self {
            Self::Multifon => Self::MULTIFON,
            Self::Emotion => Self::EMOTION,
        };
        write!(f, "{}", s)
    }
}

impl Default for API {
    fn default() -> Self {
        Self::Multifon
    }
}

impl AsRef<API> for API {
    fn as_ref(&self) -> &Self {
        self
    }
}

impl Routing {
    const GSM: &'static str = "GSM";
    const SIP: &'static str = "SIP";
    const SIP_GSM: &'static str = "SIP+GSM";
}

impl FromStr for Routing {
    type Err = String;

    fn from_str(s: &str) -> result::Result<Self, Self::Err> {
        match s.to_uppercase().as_str() {
            Self::GSM => Ok(Self::Gsm),
            Self::SIP => Ok(Self::Sip),
            Self::SIP_GSM => Ok(Self::SipGsm),
            _ => Err(format!("invalid value: {}", s)),
        }
    }
}

impl fmt::Display for Routing {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let s = match self {
            Self::Gsm => Self::GSM,
            Self::Sip => Self::SIP,
            Self::SipGsm => Self::SIP_GSM,
        };
        write!(f, "{}", s)
    }
}

impl Default for Routing {
    fn default() -> Self {
        Self::Gsm
    }
}

impl AsRef<Routing> for Routing {
    fn as_ref(&self) -> &Self {
        self
    }
}

impl fmt::Display for Status {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let s = match self {
            Self::Active => "active",
            Self::Blocked => "blocked",
        };
        write!(f, "{}", s)
    }
}

impl Default for Status {
    fn default() -> Self {
        Self::Active
    }
}

#[derive(Debug)]
pub enum Error {
    Reqwest(reqwest::Error),
    StatusCode(reqwest::StatusCode),
    Deserialization(quick_xml::DeError),
    API(response::Error),
    SetFailed(String),
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Reqwest(err) => err.fmt(f),
            Self::Deserialization(err) => err.fmt(f),
            Self::API(err) => err.fmt(f),
            Self::StatusCode(status_code) => write!(
                f,
                "{} {}",
                status_code.as_str(),
                status_code.canonical_reason().unwrap_or_default(),
            ),
            Self::SetFailed(key) => write!(f, "failed to set: {}, invalid value", key),
        }
    }
}

impl error::Error for Error {
    fn source(&self) -> Option<&(dyn error::Error + 'static)> {
        match self {
            Self::Reqwest(err) => Some(err),
            Self::Deserialization(err) => Some(err),
            Self::API(err) => Some(err),
            _ => None,
        }
    }
}

impl From<reqwest::Error> for Error {
    fn from(err: reqwest::Error) -> Self {
        Self::Reqwest(err)
    }
}

impl From<quick_xml::DeError> for Error {
    fn from(err: quick_xml::DeError) -> Self {
        Self::Deserialization(err)
    }
}

impl From<response::Error> for Error {
    fn from(err: response::Error) -> Self {
        Self::API(err)
    }
}

pub struct Client {
    login: String,
    password: String,
    api_url: &'static str,
    http_client: reqwest::Client,
}

impl Client {
    pub fn new<S: Into<String>>(
        login: S,
        password: S,
        api: Option<API>,
        http_client: Option<reqwest::Client>,
    ) -> Result<Self> {
        let api = api.unwrap_or_default();
        let http_client = match http_client {
            Some(v) => v,
            None => reqwest::ClientBuilder::new()
                .https_only(true)
                .timeout(Duration::from_secs(DEFAULT_TIMEOUT))
                .build()?,
        };
        Ok(Self {
            login: login.into(),
            password: password.into(),
            api_url: api.value(),
            http_client,
        })
    }

    pub fn get_login(&self) -> &str {
        &self.login
    }

    pub fn set_api(&mut self, api: impl AsRef<API>) {
        self.api_url = api.as_ref().value()
    }

    async fn request<T1, T2>(&self, path: &str, params: &T1) -> Result<T2>
    where
        T1: Serialize + ?Sized,
        T2: DeserializeOwned + response::XMLResultChecker,
    {
        let resp = self
            .http_client
            .post(&format!("{}/{}", self.api_url, path))
            .query(&[("login", &self.login), ("password", &self.password)])
            .query(params)
            .send()
            .await?;
        let status_code = resp.status();
        if !status_code.is_success() {
            return Err(Error::StatusCode(status_code));
        }
        let body = resp.bytes().await?;
        let res: T2 = quick_xml::de::from_reader(body.as_ref())?;
        res.check()?;
        Ok(res)
    }

    pub async fn get_balance(&self) -> Result<response::Balance> {
        let params: &[()] = &[];
        self.request("balance", params).await
    }

    pub async fn get_routing(&self) -> Result<response::Routing> {
        let params: &[()] = &[];
        self.request("routing", params).await
    }

    pub async fn set_routing(&self, routing: impl AsRef<Routing>) -> Result<()> {
        let path = "routing";
        let params = &[(path, *routing.as_ref() as i32)];
        let res: response::SetRouting = self.request(path, params).await?;
        if res.routing.is_some() {
            return Err(Error::SetFailed(path.into()));
        }
        Ok(())
    }

    pub async fn get_status(&self) -> Result<response::Status> {
        let params: &[()] = &[];
        self.request("status", params).await
    }

    pub async fn get_profile(&self) -> Result<response::Profile> {
        let params: &[()] = &[];
        self.request("profile", params).await
    }

    pub async fn get_lines(&self) -> Result<response::Lines> {
        let params: &[()] = &[];
        self.request("lines", params).await
    }

    pub async fn set_lines(&self, n: i32) -> Result<()> {
        let path = "lines";
        let params = &[(path, n)];
        let res: response::SetLines = self.request(path, params).await?;
        if res.lines.is_some() {
            return Err(Error::SetFailed(path.into()));
        }
        Ok(())
    }

    pub async fn set_password(&mut self, password: impl AsRef<str>) -> Result<()> {
        let password = password.as_ref();
        let params = &[("new_password", password)];
        self.request::<_, response::SetPassword>("password", params)
            .await?;
        self.password = password.into();
        Ok(())
    }
}
