use std::error;
use std::fmt;
use std::result;

use reqwest;
use serde::Deserialize;

use crate::client;

pub type Result<T> = result::Result<T, Error>;

#[derive(Debug)]
pub struct Error {
    pub code: i32,
    pub description: String,
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} {}", self.code, &self.description)
    }
}

impl error::Error for Error {}

#[derive(Deserialize, Debug)]
struct XMLResult {
    code: i32,
    #[serde(default)]
    description: String,
}

pub trait XMLResultChecker {
    fn check(&self) -> Result<()>;
}

impl XMLResultChecker for XMLResult {
    fn check(&self) -> Result<()> {
        if self.code as u16 == reqwest::StatusCode::OK.as_u16() {
            return Ok(());
        }
        Err(Error {
            code: self.code,
            description: self.description.clone(),
        })
    }
}

#[derive(Deserialize, Debug)]
pub struct Balance {
    result: XMLResult,
    #[serde(default)]
    pub balance: f64,
}

impl Balance {
    pub fn value(&self) -> f64 {
        self.balance
    }
}

#[derive(Deserialize, Debug)]
pub struct Routing {
    result: XMLResult,
    #[serde(default)]
    pub routing: client::Routing,
}

impl Routing {
    pub fn value(&self) -> client::Routing {
        self.routing
    }
}

#[derive(Deserialize, Debug)]
pub struct SetRouting {
    result: XMLResult,
    #[serde(default)]
    pub routing: Option<client::Routing>,
}

#[derive(Deserialize, Debug)]
pub struct Status {
    result: XMLResult,
    #[serde(default)]
    pub status: client::Status,
    #[serde(default)]
    pub expires: Option<String>,
}

impl Status {
    pub fn value(&self) -> client::Status {
        self.status
    }
}

#[derive(Deserialize, Debug)]
pub struct Profile {
    result: XMLResult,
    #[serde(default)]
    pub msisdn: String,
}

impl Profile {
    pub fn value(&self) -> &str {
        &self.msisdn
    }
}

#[derive(Deserialize, Debug)]
pub struct Lines {
    result: XMLResult,
    #[serde(rename = "ParallelCallsSipOut", default)]
    pub lines: i32,
}

impl Lines {
    pub fn value(&self) -> i32 {
        self.lines
    }
}

#[derive(Deserialize, Debug)]
pub struct SetLines {
    result: XMLResult,
    #[serde(rename = "ParallelCallsSipOut", default)]
    pub lines: Option<i32>,
}

#[derive(Deserialize, Debug)]
pub struct SetPassword {
    result: XMLResult,
}

macro_rules! impl_xml_result_checker {
    (for $($t:ty),+) => {
        $(impl XMLResultChecker for $t {
            fn check(&self) -> Result<()> {
                self.result.check()
            }
        })*
    }
}

impl_xml_result_checker!(for
    Balance,
    Routing,
    SetRouting,
    Status,
    Profile,
    Lines,
    SetLines,
    SetPassword
);
