package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	multifonapi "./lib"
)

const (
	Version = "0.0.15"

	COMMAND_BALANCE      = "balance"
	COMMAND_ROUTING      = "routing"
	COMMAND_STATUS       = "status"
	COMMAND_PROFILE      = "profile"
	COMMAND_LINES        = "lines"
	COMMAND_SET_PASSWORD = "set-password"

	EX_OK      = 0
	EX_ERR     = 1
	EX_ARG_ERR = 2 // golang flag error exit code
)

var (
	AVAILABLE_COMMANDS = [...]string{
		COMMAND_BALANCE,
		COMMAND_ROUTING,
		COMMAND_STATUS,
		COMMAND_PROFILE,
		COMMAND_LINES,
		COMMAND_SET_PASSWORD,
	}

	Login      string
	Password   string
	API        string
	Timeout    time.Duration
	Command    string
	CommandArg interface{}

	versionFlagName  = "V"
	loginFlagName    = "login"
	passwordFlagName = "password"
	apiFlagName      = "api"
	timeoutFlagName  = "timeout"

	commandMetaVar    = "COMMAND"
	commandArgMetaVar = fmt.Sprint(commandMetaVar, "_ARGUMENT")
)

func getAPIs() []string {
	res := make([]string, len(multifonapi.API_NAME_URL_MAP))
	i := 0
	for k := range multifonapi.API_NAME_URL_MAP {
		res[i] = k
		i++
	}
	return res
}

func getRoutingDescriptions() []string {
	res := make([]string, len(multifonapi.ROUTING_DESCRIPTION_MAP))
	i := 0
	for _, v := range multifonapi.ROUTING_DESCRIPTION_MAP {
		res[i] = v
		i++
	}
	return res
}

func getRoutingByDescription(s string) int {
	for k, v := range multifonapi.ROUTING_DESCRIPTION_MAP {
		if v == s {
			return k
		}
	}
	return -1
}

func printUsage() {
	var name string
	if len(os.Args) < 1 {
		name = "PROG_NAME"
	} else {
		name = filepath.Base(os.Args[0])
	}
	namePadding := strings.Repeat(" ", len(name))
	helpFlagName := "h"
	passwordMetaVar := "PASSWORD"
	apiMetaVar := "API"
	timeoutMetaVar := "TIMEOUT"
	apis := getAPIs()
	sort.Strings(apis)
	routingDescriptions := getRoutingDescriptions()
	sort.Strings(routingDescriptions)
	choicesDelimiter := " | "
	fmt.Fprintf(
		flag.CommandLine.Output(),
		"%s [-%s%s] -%s <LOGIN> -%s <%s>\n"+
			"%s [-%s <%s>] [-%s <%s>]\n"+
			"%s <%s> [<%s>]\n\n"+
			"[-%s] * Print help and exit\n"+
			"[-%s] * Print version and exit\n\n"+
			"%s:\n"+
			"  { %s } (default: %s)\n\n"+
			"%s:\n"+
			"  time.ParseDuration (default: %s)\n\n"+
			"%s:\n"+
			"  { %s }\n\n"+
			"%s:\n"+
			"  %s { %s }\n"+
			"  %s <NUMBER> (2 .. 20)\n"+
			"  %s <NEW_%s> (min 8, max 20, mixed case, digits)\n",
		name, helpFlagName, versionFlagName, loginFlagName, passwordFlagName,
		passwordMetaVar, namePadding, apiFlagName, apiMetaVar, timeoutFlagName,
		timeoutMetaVar, namePadding, commandMetaVar, commandArgMetaVar,
		helpFlagName, versionFlagName, apiMetaVar,
		strings.Join(apis, choicesDelimiter), multifonapi.API_DEFAULT,
		timeoutMetaVar, multifonapi.DEFAULT_TIMEOUT, commandMetaVar,
		strings.Join(AVAILABLE_COMMANDS[:], choicesDelimiter),
		commandArgMetaVar, COMMAND_ROUTING,
		strings.Join(routingDescriptions, choicesDelimiter), COMMAND_LINES,
		COMMAND_SET_PASSWORD, passwordMetaVar,
	)
}

func flagNameToFlag(name string) string {
	return fmt.Sprint("-", name)
}

func fatalParseArgs(k, v string) {
	fmt.Fprintf(os.Stderr, "Failed to parse argument %s: \"%s\"\n", k, v)
	os.Exit(EX_ARG_ERR)
}

func parseAPI() {
	API = strings.ToLower(API)
	if _, ok := multifonapi.API_NAME_URL_MAP[API]; !ok {
		fatalParseArgs(flagNameToFlag(apiFlagName), API)
	}
}

func parseCommand() {
	Command = strings.ToLower(flag.Arg(0))
	for _, v := range AVAILABLE_COMMANDS {
		if v == Command {
			return
		}
	}
	fatalParseArgs(commandMetaVar, Command)
}

func parseCommandArg() {
	arg := flag.Arg(1)
	switch Command {
	case COMMAND_ROUTING:
		if arg == "" {
			return
		}
		routing := getRoutingByDescription(strings.ToUpper(arg))
		if routing == -1 {
			fatalParseArgs(commandArgMetaVar, arg)
		}
		CommandArg = routing
	case COMMAND_LINES:
		if arg == "" {
			return
		}
		n, err := strconv.Atoi(arg)
		if err != nil {
			fatalParseArgs(commandArgMetaVar, arg)
		}
		CommandArg = n
	case COMMAND_SET_PASSWORD:
		if arg == "" {
			fatalParseArgs(commandArgMetaVar, arg)
		}
		CommandArg = arg
	}
}

func parseArgs() {
	flag.Usage = printUsage
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(EX_ARG_ERR)
	}
	isVersion := flag.Bool(versionFlagName, false, "")
	flag.StringVar(&Login, loginFlagName, "", "")
	flag.StringVar(&Password, passwordFlagName, "", "")
	flag.StringVar(&API, apiFlagName, multifonapi.API_DEFAULT, "")
	flag.DurationVar(
		&Timeout,
		timeoutFlagName,
		multifonapi.DEFAULT_TIMEOUT,
		"",
	)
	flag.Parse()
	if *isVersion {
		fmt.Println(Version)
		os.Exit(EX_OK)
	}
	if Login == "" {
		fatalParseArgs(flagNameToFlag(loginFlagName), Login)
	}
	if Password == "" {
		fatalParseArgs(flagNameToFlag(passwordFlagName), Password)
	}
	parseAPI()
	parseCommand()
	parseCommandArg()
}

func main() {
	parseArgs()
	client := multifonapi.NewClient(
		Login,
		Password,
		API,
		&http.Client{Timeout: Timeout},
	)
	fatalIfErr := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(EX_ERR)
		}
	}
	strOk := "OK"
	switch Command {
	case COMMAND_BALANCE:
		res, err := client.GetBalance()
		fatalIfErr(err)
		fmt.Println(res.Balance)
	case COMMAND_ROUTING:
		if CommandArg == nil {
			res, err := client.GetRouting()
			fatalIfErr(err)
			val := res.Description()
			if val == "" {
				fmt.Println(res.Routing)
			} else {
				fmt.Println(val)
			}
		} else {
			_, err := client.SetRouting(CommandArg.(int))
			fatalIfErr(err)
			fmt.Println(strOk)
		}
	case COMMAND_STATUS:
		res, err := client.GetStatus()
		fatalIfErr(err)
		val := res.Description()
		if val == "" {
			val = strconv.Itoa(res.Status)
		}
		if res.Expires != "" {
			val = fmt.Sprintf("%s:%s", val, res.Expires)
		}
		fmt.Println(val)
	case COMMAND_PROFILE:
		res, err := client.GetProfile()
		fatalIfErr(err)
		fmt.Println(res.MSISDN)
	case COMMAND_LINES:
		if CommandArg == nil {
			res, err := client.GetLines()
			fatalIfErr(err)
			fmt.Println(res.Lines)
		} else {
			_, err := client.SetLines(CommandArg.(int))
			fatalIfErr(err)
			fmt.Println(strOk)
		}
	case COMMAND_SET_PASSWORD:
		_, err := client.SetPassword(CommandArg.(string))
		fatalIfErr(err)
		fmt.Println(strOk)
	}
}
