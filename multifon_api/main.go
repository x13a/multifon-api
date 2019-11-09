package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	multifonapi "./lib"
)

const (
	Version = "0.0.2"

	API_MULTIFON = "multifon"
	API_EMOTION  = "emotion"

	COMMAND_BALANCE      = "balance"
	COMMAND_GET_ROUTING  = "get-routing"
	COMMAND_SET_ROUTING  = "set-routing"
	COMMAND_STATUS       = "status"
	COMMAND_PROFILE      = "profile"
	COMMAND_GET_LINES    = "get-lines"
	COMMAND_SET_LINES    = "set-lines"
	COMMAND_SET_PASSWORD = "set-password"

	EX_OK      = 0
	EX_ERR     = 1
	EX_ARG_ERR = 2 // golang flag error exit code
)

var (
	API_URL_MAP = map[string]string{
		API_MULTIFON: multifonapi.MULTIFON_API_URL,
		API_EMOTION:  multifonapi.EMOTION_API_URL,
	}

	AVAILABLE_COMMANDS = [...]string{
		COMMAND_BALANCE,
		COMMAND_GET_ROUTING,
		COMMAND_SET_ROUTING,
		COMMAND_STATUS,
		COMMAND_PROFILE,
		COMMAND_GET_LINES,
		COMMAND_SET_LINES,
		COMMAND_SET_PASSWORD,
	}

	Login      string
	Password   string
	API        string
	Command    string
	CommandArg interface{}

	versionFlagName  = "V"
	loginFlagName    = "login"
	passwordFlagName = "password"
	apiFlagName      = "api"

	commandMetaVar    = "COMMAND"
	commandArgMetaVar = fmt.Sprint(commandMetaVar, "_ARGUMENT")
)

func getDefaultAPI() string {
	for k, v := range API_URL_MAP {
		if v == multifonapi.DEFAULT_API_URL {
			return k
		}
	}
	return ""
}

func getAPIs() []string {
	res := make([]string, len(API_URL_MAP))
	i := 0
	for k := range API_URL_MAP {
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
	name := path.Base(os.Args[0])
	helpFlagName := "h"
	passwordMetaVar := "PASSWORD"
	apiMetaVar := "API"
	apis := getAPIs()
	sort.Strings(apis)
	routingDescriptions := getRoutingDescriptions()
	sort.Strings(routingDescriptions)
	choicesDelimiter := " | "
	fmt.Fprintf(
		flag.CommandLine.Output(),
		"%s [-%s] [-%s] -%s <LOGIN> -%s <%s> [-%s <%s>] \n"+
			"%s <%s> [<%s>]\n\n"+
			"[-%s] * Print help and exit\n"+
			"[-%s] * Print version and exit\n\n"+
			"%s:\n"+
			"  { %s } (default: %s)\n\n"+
			"%s:\n"+
			"  { %s }\n\n"+
			"%s:\n"+
			"  %s { %s }\n"+
			"  %s <NUMBER> (2 .. 20)\n"+
			"  %s <NEW_%s> (min 8, max 20, mixed case, digits)\n",
		name, helpFlagName, versionFlagName, loginFlagName, passwordFlagName,
		passwordMetaVar, apiFlagName, apiMetaVar,
		strings.Repeat(" ", len(name)), commandMetaVar, commandArgMetaVar,
		helpFlagName, versionFlagName, apiMetaVar,
		strings.Join(apis, choicesDelimiter), getDefaultAPI(), commandMetaVar,
		strings.Join(AVAILABLE_COMMANDS[:], choicesDelimiter),
		commandArgMetaVar, COMMAND_SET_ROUTING,
		strings.Join(routingDescriptions, choicesDelimiter), COMMAND_SET_LINES,
		COMMAND_SET_PASSWORD, passwordMetaVar,
	)
}

func fatalParseArgs(k, v string) {
	fmt.Fprintf(os.Stderr, "failed to parse argument %s: \"%s\"\n", k, v)
	os.Exit(EX_ARG_ERR)
}

func flagNameToFlag(s string) string {
	return fmt.Sprint("-", s)
}

func fatalIfEmptyArg(desc, val string) {
	if val == "" {
		fatalParseArgs(desc, val)
	}
}

func parseAPI() {
	api, ok := API_URL_MAP[strings.ToLower(API)]
	if !ok {
		fatalParseArgs(flagNameToFlag(apiFlagName), API)
	}
	API = api
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
	case COMMAND_SET_ROUTING:
		routing := getRoutingByDescription(strings.ToUpper(arg))
		if routing == -1 {
			fatalParseArgs(commandArgMetaVar, arg)
		}
		CommandArg = routing
	case COMMAND_SET_LINES:
		n, err := strconv.Atoi(arg)
		if err != nil {
			fatalParseArgs(commandArgMetaVar, arg)
		}
		CommandArg = n
	case COMMAND_SET_PASSWORD:
		fatalIfEmptyArg(commandArgMetaVar, arg)
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
	flag.StringVar(&API, apiFlagName, getDefaultAPI(), "")
	flag.Parse()
	if *isVersion {
		fmt.Println(Version)
		os.Exit(EX_OK)
	}
	fatalIfEmptyArg(flagNameToFlag(loginFlagName), Login)
	fatalIfEmptyArg(flagNameToFlag(passwordFlagName), Password)
	parseAPI()
	parseCommand()
	parseCommandArg()
}

func main() {
	parseArgs()
	client := multifonapi.NewClient(Login, Password, API, nil)
	fatalIfErr := func(e error) {
		if e != nil {
			fmt.Println(e)
			os.Exit(EX_ERR)
		}
	}
	strOk := "OK"
	switch Command {
	case COMMAND_BALANCE:
		res, err := client.GetBalance()
		fatalIfErr(err)
		fmt.Println(res.Balance)
	case COMMAND_GET_ROUTING:
		res, err := client.GetRouting()
		fatalIfErr(err)
		fmt.Println(res.Description())
	case COMMAND_SET_ROUTING:
		_, err := client.SetRouting(CommandArg.(int))
		fatalIfErr(err)
		fmt.Println(strOk)
	case COMMAND_STATUS:
		res, err := client.GetStatus()
		fatalIfErr(err)
		val := res.Description()
		if res.Expires != "" {
			val = fmt.Sprint(val, "|", res.Expires)
		}
		fmt.Println(val)
	case COMMAND_PROFILE:
		res, err := client.GetProfile()
		fatalIfErr(err)
		fmt.Println(res.MSISDN)
	case COMMAND_GET_LINES:
		res, err := client.GetLines()
		fatalIfErr(err)
		fmt.Println(res.Lines)
	case COMMAND_SET_LINES:
		_, err := client.SetLines(CommandArg.(int))
		fatalIfErr(err)
		fmt.Println(strOk)
	case COMMAND_SET_PASSWORD:
		_, err := client.SetPassword(CommandArg.(string))
		fatalIfErr(err)
		fmt.Println(strOk)
	}
}
