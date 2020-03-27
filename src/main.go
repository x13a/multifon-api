package main

import (
	"errors"
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

type API multifonapi.API

func (a *API) String() string {
	return string(*a)
}

func (a *API) Set(s string) error {
	api := multifonapi.API(strings.ToLower(s))
	if _, ok := multifonapi.APIUrlMap[api]; ok {
		*a = API(api)
		return nil
	}
	return errors.New("parse error")
}

func (a *API) Unwrap() multifonapi.API {
	return multifonapi.API(*a)
}

const (
	FlagHelp     = "h"
	FlagVersion  = "V"
	FlagLogin    = "login"
	FlagPassword = "password"
	FlagAPI      = "api"
	FlagTimeout  = "timeout"

	MetaVarLogin      = "LOGIN"
	MetaVarPassword   = "PASSWORD"
	MetaVarAPI        = "API"
	MetaVarTimeout    = "TIMEOUT"
	MetaVarCommand    = "COMMAND"
	MetaVarCommandArg = "COMMAND_ARGUMENT"

	CommandBalance     = "balance"
	CommandRouting     = "routing"
	CommandStatus      = "status"
	CommandProfile     = "profile"
	CommandLines       = "lines"
	CommandSetPassword = "set-password"

	ExOk     = 0
	ExErr    = 1
	ExArgErr = 2 // golang flag error exit code
)

var Commands = [...]string{
	CommandBalance,
	CommandRouting,
	CommandStatus,
	CommandProfile,
	CommandLines,
	CommandSetPassword,
}

func getAPIChoices() []string {
	res := make([]string, len(multifonapi.APIUrlMap))
	i := 0
	for k := range multifonapi.APIUrlMap {
		res[i] = k.String()
		i++
	}
	return res
}

func getRoutingDescriptions() []string {
	res := make([]string, len(multifonapi.RoutingDescriptionMap))
	i := 0
	for _, v := range multifonapi.RoutingDescriptionMap {
		res[i] = v
		i++
	}
	return res
}

func getRoutingByDescription(s string) multifonapi.Routing {
	for k, v := range multifonapi.RoutingDescriptionMap {
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
	indention := strings.Repeat(" ", len(name))
	apiChoices := getAPIChoices()
	sort.Strings(apiChoices)
	routingDescriptions := getRoutingDescriptions()
	sort.Strings(routingDescriptions)
	sep := " | "
	fmt.Fprintf(
		flag.CommandLine.Output(),
		"%s [-%s%s] -%s <%s> -%s <%s>\n"+
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
		name, FlagHelp, FlagVersion, FlagLogin, MetaVarLogin, FlagPassword,
		MetaVarPassword, indention, FlagAPI, MetaVarAPI, FlagTimeout,
		MetaVarTimeout, indention, MetaVarCommand, MetaVarCommandArg,
		FlagHelp, FlagVersion, MetaVarAPI, strings.Join(apiChoices, sep),
		multifonapi.APIDefault, MetaVarTimeout, multifonapi.DefaultTimeout,
		MetaVarCommand, strings.Join(Commands[:], sep), MetaVarCommandArg,
		CommandRouting, strings.Join(routingDescriptions, sep), CommandLines,
		CommandSetPassword, MetaVarPassword,
	)
}

func flagNameToFlag(name string) string {
	return fmt.Sprint("-", name)
}

func fatalParseArgs(k, v string) {
	fmt.Fprintf(os.Stderr, "Failed to parse argument %s: \"%s\"\n", k, v)
	os.Exit(ExArgErr)
}

func parseCommand(opts *Opts) {
	cmd := strings.ToLower(flag.Arg(0))
	for _, v := range Commands {
		if v == cmd {
			opts.command = cmd
			return
		}
	}
	fatalParseArgs(MetaVarCommand, cmd)
}

func parseCommandArg(opts *Opts) {
	arg := flag.Arg(1)
	switch opts.command {
	case CommandRouting:
		if arg == "" {
			return
		}
		routing := getRoutingByDescription(strings.ToUpper(arg))
		if routing == -1 {
			fatalParseArgs(MetaVarCommandArg, arg)
		}
		opts.commandArg = routing
	case CommandLines:
		if arg == "" {
			return
		}
		n, err := strconv.Atoi(arg)
		if err != nil {
			fatalParseArgs(MetaVarCommandArg, arg)
		}
		opts.commandArg = n
	case CommandSetPassword:
		if arg == "" {
			fatalParseArgs(MetaVarCommandArg, arg)
		}
		opts.commandArg = arg
	}
}

type Opts struct {
	login      string
	password   string
	api        API
	timeout    time.Duration
	command    string
	commandArg interface{}
}

func parseArgs() *Opts {
	flag.Usage = printUsage
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(ExArgErr)
	}
	opts := &Opts{}
	isHelp := flag.Bool(FlagHelp, false, "")
	isVersion := flag.Bool(FlagVersion, false, "")
	flag.StringVar(&opts.login, FlagLogin, "", "")
	flag.StringVar(&opts.password, FlagPassword, "", "")
	flag.Var(&opts.api, FlagAPI, "")
	flag.DurationVar(
		&opts.timeout,
		FlagTimeout,
		multifonapi.DefaultTimeout,
		"",
	)
	flag.Parse()
	if *isHelp {
		flag.Usage()
		os.Exit(ExOk)
	}
	if *isVersion {
		fmt.Println(multifonapi.Version)
		os.Exit(ExOk)
	}
	if opts.login == "" {
		fatalParseArgs(flagNameToFlag(FlagLogin), opts.login)
	}
	if opts.password == "" {
		fatalParseArgs(flagNameToFlag(FlagPassword), opts.password)
	}
	parseCommand(opts)
	parseCommandArg(opts)
	return opts
}

func main() {
	opts := parseArgs()
	client := multifonapi.NewClient(
		opts.login,
		opts.password,
		opts.api.Unwrap(),
		&http.Client{Timeout: opts.timeout},
	)
	fatalIfErr := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(ExErr)
		}
	}
	strOk := "OK"
	switch opts.command {
	case CommandBalance:
		res, err := client.GetBalance()
		fatalIfErr(err)
		fmt.Println(res.Balance)
	case CommandRouting:
		if opts.commandArg == nil {
			res, err := client.GetRouting()
			fatalIfErr(err)
			val := res.Description()
			if val == "" {
				fmt.Println(res.Routing)
			} else {
				fmt.Println(val)
			}
		} else {
			_, err := client.SetRouting(opts.commandArg.(multifonapi.Routing))
			fatalIfErr(err)
			fmt.Println(strOk)
		}
	case CommandStatus:
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
	case CommandProfile:
		res, err := client.GetProfile()
		fatalIfErr(err)
		fmt.Println(res.MSISDN)
	case CommandLines:
		if opts.commandArg == nil {
			res, err := client.GetLines()
			fatalIfErr(err)
			fmt.Println(res.Lines)
		} else {
			_, err := client.SetLines(opts.commandArg.(int))
			fatalIfErr(err)
			fmt.Println(strOk)
		}
	case CommandSetPassword:
		_, err := client.SetPassword(opts.commandArg.(string))
		fatalIfErr(err)
		fmt.Println(strOk)
	}
}
