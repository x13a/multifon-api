# multifon-api

[Multifon API](https://multifon.megafon.ru/)

## Installation
```sh
$ make
$ make install
```
or
```sh
$ brew tap x31a/tap https://bitbucket.org/x31a/homebrew-tap.git
$ brew install x31a/tap/multifon-api
```

## Usage
```text
multifon-api [-hV] -login <LOGIN> -password <PASSWORD>
             [-api <API>] [-timeout <TIMEOUT>]
             <COMMAND> [<COMMAND_ARGUMENT>]

[-h] * Print help and exit
[-V] * Print version and exit

API:
  { emotion | multifon } (default: multifon)

TIMEOUT:
  time.ParseDuration (default: 30s)

COMMAND:
  { balance | routing | status | profile | lines | set-password }

COMMAND_ARGUMENT:
  routing { GSM | SIP | SIP+GSM }
  lines <NUMBER> (2 .. 20)
  set-password <NEW_PASSWORD> (min 8, max 20, mixed case, digits)
```

## Example

To get balance:
```sh
$ multifon-api -login "LOGIN" -password "PASSWORD" balance
```

To set routing:
```sh
$ multifon-api -login "LOGIN" -password "PASSWORD" routing gsm
```
