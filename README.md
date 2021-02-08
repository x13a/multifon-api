# multifon-api

[Multifon API](https://multifon.megafon.ru/)

## Installation
```sh
$ make
$ make install
```
or
```sh
$ brew tap x31a/tap
$ brew install x31a/tap/multifon-api
```

## Security

Don't forget to set right permissions for configuration file:
```sh
$ chmod 600 ./config/multifon.json
```

## Usage
```text
multifon 0.1.1

USAGE:
    multifon [OPTIONS] <SUBCOMMAND>

FLAGS:
    -h, --help       Prints help information
    -V, --version    Prints version information

OPTIONS:
    -a, --api <api>              (env: MULTIFON_API)
    -c, --config <config>
    -l, --login <login>          (env: MULTIFON_LOGIN)
    -p, --password <password>    (env: MULTIFON_PASSWORD)
    -t, --timeout <timeout>      (env: MULTIFON_TIMEOUT)

SUBCOMMANDS:
    balance
    help            Prints this message or the help of the given subcommand(s)
    lines           get / set
    profile
    routing         get / set
    set-password    (env: MULTIFON_NEW_PASSWORD)
    status
```

## Example

To get balance:
```sh
$ multifon --config ~/multifon.json balance
```

To set routing:
```sh
$ multifon --config ~/multifon.json routing gsm
```

To get status (stdin config):
```sh
$ cat ~/multifon.json | multifon --config - status
```

To set lines (env identity, **space before first variable!**):
```sh
$  MULTIFON_LOGIN="login" MULTIFON_PASSWORD="password" multifon lines 2
```

## Library
```rust
use std::error;

use multifon::Client;

#[tokio::main]
async fn main() -> Result<(), Box<dyn error::Error>> {
    let client = Client::new("LOGIN", "PASSWORD", None, None)?;
    println!("{}", client.get_balance().await?.value());
    Ok(())
}
```
