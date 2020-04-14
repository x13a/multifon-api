package main

import (
	"encoding/json"
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

const (
	FlagHelp     = "h"
	FlagVersion  = "V"
	FlagConfig   = "config"
	FlagLogin    = "login"
	FlagPassword = "password"
	FlagAPI      = "api"
	FlagTimeout  = "timeout"

	MetaVarConfig     = "CONFIG"
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

	EnvLogin       = "MULTIFON_LOGIN"
	EnvPassword    = "MULTIFON_PASSWORD"
	EnvNewPassword = "MULTIFON_NEW_PASSWORD"

	ArgStdin = "-"
)

var Commands = [...]string{
	CommandBalance,
	CommandRouting,
	CommandStatus,
	CommandProfile,
	CommandLines,
	CommandSetPassword,
}

type Config struct {
	Login       string   `json:"login,omitempty"`
	Password    string   `json:"password"`
	NewPassword string   `json:"new_password"`
	API         API      `json:"api,omitempty"`
	Timeout     *Timeout `json:"timeout,omitempty"`
	path        string
}

func (c Config) String() string {
	return ""
}

func (c *Config) Set(s string) error {
	var file *os.File
	var err error
	if s == ArgStdin {
		file = os.Stdin
	} else {
		file, err = os.Open(s)
		if err != nil {
			return err
		}
		defer file.Close()
		c.path = s
	}
	return json.NewDecoder(file).Decode(c)
}

type (
	Timeout time.Duration
	API     multifonapi.API
)

func (t Timeout) String() string {
	return t.unwrap().String()
}

func (t *Timeout) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*t = Timeout(v)
	return nil
}

func (t Timeout) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Timeout) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return t.Set(s)
}

func (t Timeout) unwrap() time.Duration {
	return time.Duration(t)
}

func (a API) String() string {
	return string(a)
}

func (a *API) Set(s string) error {
	api := multifonapi.API(strings.ToLower(s))
	if _, ok := multifonapi.APIUrlMap[api]; ok {
		*a = API(api)
		return nil
	}
	return errors.New("api parse error")
}

func (a API) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *API) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return a.Set(s)
}

func (a API) unwrap() multifonapi.API {
	return multifonapi.API(a)
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

func format(s string, m map[string]interface{}) string {
	args := make([]string, len(m)*2)
	i := 0
	for k, v := range m {
		args[i] = fmt.Sprintf("{%s}", k)
		args[i+1] = fmt.Sprint(v)
		i += 2
	}
	return strings.NewReplacer(args...).Replace(s)
}

func printUsage() {
	var name string
	if len(os.Args) < 1 {
		name = "PROG_NAME"
	} else {
		name = filepath.Base(os.Args[0])
	}
	apiChoices := getAPIChoices()
	sort.Strings(apiChoices)
	routingDescriptions := getRoutingDescriptions()
	sort.Strings(routingDescriptions)
	sep := " | "
	fmt.Fprintln(
		flag.CommandLine.Output(),
		format(
			"{NAME} [-{h}{V}] ( -{c} <{C}> | -{l} <{L}> -{p} <{P}> )\n"+
				"{TAB} [-{a} <{A}>] [-{t} <{T}>] <{CMD}> [<{CMDARG}>]\n\n"+
				"[-{h}] * Print help and exit\n"+
				"[-{V}] * Print version and exit\n\n"+
				"{C}:\n"+
				"  JSON filepath\n"+
				"    + fields: [{l}, {p}, new_{p}, {a}, {t}]\n"+
				"    + stdin:  {STDIN}\n\n"+
				"{L}:\n"+
				"  string (env: {ENVL})\n\n"+
				"{P}:\n"+
				"  string (env: {ENVP})\n\n"+
				"{A}:\n"+
				"  { {CA} } (default: {DEFA})\n\n"+
				"{T}:\n"+
				"  time.ParseDuration (default: {DEFT})\n\n"+
				"{CMD}:\n"+
				"  { {CCMD} }\n\n"+
				"{CMDARG}:\n"+
				"  {CMDR} { {CCMDR} }\n"+
				"  {CMDL} <NUMBER> (2 .. 20)\n"+
				"  {CMDSP} <NEW_{P}>\n"+
				"\ttip: min 8, max 20, mixed case, digits\n"+
				"\tenv: {ENVNP}",
			map[string]interface{}{
				"NAME":   name,
				"TAB":    strings.Repeat(" ", len(name)),
				"h":      FlagHelp,
				"V":      FlagVersion,
				"c":      FlagConfig,
				"l":      FlagLogin,
				"p":      FlagPassword,
				"a":      FlagAPI,
				"t":      FlagTimeout,
				"C":      MetaVarConfig,
				"L":      MetaVarLogin,
				"P":      MetaVarPassword,
				"A":      MetaVarAPI,
				"T":      MetaVarTimeout,
				"CMD":    MetaVarCommand,
				"CMDARG": MetaVarCommandArg,
				"STDIN":  ArgStdin,
				"DEFA":   multifonapi.DefaultAPI,
				"DEFT":   multifonapi.DefaultTimeout,
				"ENVL":   EnvLogin,
				"ENVP":   EnvPassword,
				"ENVNP":  EnvNewPassword,
				"CA":     strings.Join(apiChoices, sep),
				"CCMD":   strings.Join(Commands[:], sep),
				"CCMDR":  strings.Join(routingDescriptions, sep),
				"CMDR":   CommandRouting,
				"CMDL":   CommandLines,
				"CMDSP":  CommandSetPassword,
			},
		),
	)
}

func flagNameToFlag(name string) string {
	return fmt.Sprint("-", name)
}

func fatalParseArgs(k, v string) {
	fmt.Fprintf(os.Stderr, "Failed to parse argument %s: `%s`\n", k, v)
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
		if !parseIdentity(&arg, opts.config.NewPassword, EnvNewPassword) {
			fatalParseArgs(MetaVarCommandArg, arg)
		}
		opts.commandArg = arg
	}
}

func parseIdentity(arg *string, configValue, envKey string) bool {
	if *arg == "" {
		value := configValue
		if value == "" {
			value = os.Getenv(envKey)
			if value == "" {
				return false
			}
		}
		*arg = value
	}
	return true
}

type Opts struct {
	config     Config
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
	flag.Var(&opts.config, FlagConfig, "")
	flag.StringVar(&opts.login, FlagLogin, "", "")
	flag.StringVar(&opts.password, FlagPassword, "", "")
	flag.Var(&opts.api, FlagAPI, "")
	flag.DurationVar(&opts.timeout, FlagTimeout, -1, "")
	flag.Parse()
	if *isHelp {
		flag.Usage()
		os.Exit(ExOk)
	}
	if *isVersion {
		fmt.Println(multifonapi.Version)
		os.Exit(ExOk)
	}
	if !parseIdentity(&opts.login, opts.config.Login, EnvLogin) {
		fatalParseArgs(flagNameToFlag(FlagLogin), opts.login)
	}
	if !parseIdentity(&opts.password, opts.config.Password, EnvPassword) {
		fatalParseArgs(flagNameToFlag(FlagPassword), opts.password)
	}
	parseCommand(opts)
	parseCommandArg(opts)
	if opts.api == "" {
		opts.api = opts.config.API
	}
	if opts.timeout < 0 && opts.config.Timeout != nil {
		opts.timeout = opts.config.Timeout.unwrap()
	}
	return opts
}

func updateConfigFile(opts *Opts) error {
	file, err := os.OpenFile(opts.config.path, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	opts.config.Password = opts.commandArg.(string)
	opts.config.NewPassword = opts.password
	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")
	return enc.Encode(opts.config)
}

func main() {
	opts := parseArgs()
	var httpClient *http.Client
	if opts.timeout >= 0 {
		httpClient = &http.Client{Timeout: opts.timeout}
	}
	client := multifonapi.NewClient(
		opts.login,
		opts.password,
		opts.api.unwrap(),
		httpClient,
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
		if opts.config.path != "" {
			fatalIfErr(updateConfigFile(opts))
		}
		fmt.Println(strOk)
	}
}
