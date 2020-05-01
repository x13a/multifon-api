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
multifon-api [-hV] ( -config <CONFIG> | -login <LOGIN> -password <PASSWORD> )
             [-api <API>] [-timeout <TIMEOUT>] <COMMAND> [<COMMAND_ARGUMENT>]

[-h] * Print help and exit
[-V] * Print version and exit

CONFIG:
  filepath (stdin: -)

LOGIN:
  string (env: MULTIFON_LOGIN)

PASSWORD:
  string (env: MULTIFON_PASSWORD)

API:
  { emotion | multifon } (default: multifon)

TIMEOUT:
  time.ParseDuration (default: 32s)

COMMAND:
  { balance | routing | status | profile | lines | set-password }

COMMAND_ARGUMENT:
  routing { GSM | SIP | SIP+GSM }
  lines <NUMBER> (2 .. 20)
  set-password <NEW_PASSWORD>
	tip: min 8, max 20, mixed case, digits
	env: MULTIFON_NEW_PASSWORD
```

## Example

To get balance:
```sh
$ multifon-api -config ~/multifon-api.json balance
```

To set routing:
```sh
$ multifon-api -config ~/multifon-api.json routing gsm
```

To get status (stdin config):
```sh
$ cat ~/multifon-api.json | multifon-api -config - status
```

To set lines (env identity, **space before first variable!**):
```sh
$  MULTIFON_LOGIN="login" MULTIFON_PASSWORD="password" multifon-api lines 2
```
