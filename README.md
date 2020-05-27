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
multifon [-hV] ( -config <CONFIG> | -login <LOGIN> -password <PASSWORD> )
         [-api <API>] [-timeout <TIMEOUT>] <COMMAND> [<COMMAND_ARGUMENT>]

[-h] * Print help and exit
[-V] * Print version and exit

-c, -config:
  filepath (stdin: -)

-l, -login:
  string (env: MULTIFON_LOGIN)

-p, -password:
  string (env: MULTIFON_PASSWORD)

-a, -api:
  { emotion | multifon } (default: multifon, env: MULTIFON_API)

-t, -timeout:
  time.ParseDuration (default: 32s, env: MULTIFON_TIMEOUT)

COMMAND:
  { balance | routing | status | profile | lines | set-password }

COMMAND_ARGUMENT:
  routing { GSM | SIP | SIP+GSM }
  lines <NUMBER> (2 .. 20)
  set-password <NEW_PASSWORD>
      tip: 8 <= x <= 20, mixed case, digits
      env: MULTIFON_NEW_PASSWORD
```

## Example

To get balance:
```sh
$ multifon -config ~/multifon.json balance
```

To set routing:
```sh
$ multifon -config ~/multifon.json routing gsm
```

To get status (stdin config):
```sh
$ cat ~/multifon.json | multifon -config - status
```

To set lines (env identity, **space before first variable!**):
```sh
$  MULTIFON_LOGIN="login" MULTIFON_PASSWORD="password" multifon lines 2
```

## Library
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"bitbucket.org/x31a/multifon-api/src/multifon"
)

func main() {
	login := "login"
	password := "password"

	// Default client
	client := multifon.NewClient(login, password, "", nil)

	// Requesting balance
	res, err := client.GetBalance(context.Background())
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(res.Balance)

	// Custom client
	client = multifon.NewClient(
		login,
		password,
		multifon.APIEmotion,
		&http.Client{Timeout: 5 * time.Second},
	)

	// Setting routing
	_, err = client.SetRouting(context.Background(), multifon.RoutingGSM)
	if err != nil {
		log.Fatalln(err)
	}

	// Switching api
	client.SetAPI(multifon.APIMultifon)
}
```
