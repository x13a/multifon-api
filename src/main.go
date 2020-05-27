package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"./multifon"
)

const (
	CommandBalance     = "balance"
	CommandRouting     = "routing"
	CommandStatus      = "status"
	CommandProfile     = "profile"
	CommandLines       = "lines"
	CommandSetPassword = "set-password"

	envPrefix      = "MULTIFON_"
	EnvLogin       = envPrefix + "LOGIN"
	EnvPassword    = envPrefix + "PASSWORD"
	EnvAPI         = envPrefix + "API"
	EnvTimeout     = envPrefix + "TIMEOUT"
	EnvNewPassword = envPrefix + "NEW_PASSWORD"

	MetaVarCommand    = "COMMAND"
	MetaVarCommandArg = MetaVarCommand + "_ARGUMENT"

	ExitSuccess = 0
	ExitFailure = 1
	ExitUsage   = 2

	ArgStdin = "-"
)

var (
	FlagHelp     = MakeFlag("", "help")
	FlagVersion  = MakeFlag("V", "version")
	FlagConfig   = MakeFlag("", "config")
	FlagLogin    = MakeFlag("", "login")
	FlagPassword = MakeFlag("", "password")
	FlagAPI      = MakeFlag("", "api")
	FlagTimeout  = MakeFlag("", "timeout")

	Commands = [...]string{
		CommandBalance,
		CommandRouting,
		CommandStatus,
		CommandProfile,
		CommandLines,
		CommandSetPassword,
	}
)

type Flag struct {
	ShortName string
	LongName  string
}

func (f Flag) Names() []string {
	return []string{f.ShortName, f.LongName}
}

func (f Flag) MetaVar() string {
	return strings.ToUpper(f.LongName)
}

func MakeFlag(short, long string) Flag {
	if short == "" {
		short = long[:1]
	}
	return Flag{short, long}
}

type Config struct {
	Login       string    `json:"login,omitempty"`
	Password    string    `json:"password"`
	NewPassword string    `json:"new_password"`
	API         API       `json:"api,omitempty"`
	Timeout     *Duration `json:"timeout,omitempty"`
	path        string
}

func (c *Config) String() string {
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
	Duration time.Duration
	API      multifon.API
)

func (d Duration) String() string {
	return d.Unwrap().String()
}

func (d *Duration) Set(s string) error {
	if s == "" {
		*d = Duration(multifon.DefaultTimeout)
		return nil
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return d.Set(s)
}

func (d Duration) Unwrap() time.Duration {
	return time.Duration(d)
}

func (a API) String() string {
	return string(a)
}

func (a *API) Set(s string) error {
	if s == "" {
		*a = API(multifon.DefaultAPI)
		return nil
	}
	api := multifon.API(strings.ToLower(s))
	if _, ok := multifon.APIUrlMap[api]; ok {
		*a = API(api)
		return nil
	}
	return fmt.Errorf("<%s> parse error", FlagAPI.MetaVar())
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

func (a API) Unwrap() multifon.API {
	return multifon.API(a)
}

func getAPIChoices() []string {
	res := make([]string, len(multifon.APIUrlMap))
	i := 0
	for k := range multifon.APIUrlMap {
		res[i] = string(k)
		i++
	}
	return res
}

func getRoutingDescriptions() []string {
	res := make([]string, len(multifon.RoutingDescriptionMap))
	i := 0
	for _, v := range multifon.RoutingDescriptionMap {
		res[i] = v
		i++
	}
	return res
}

func getRoutingByDescription(s string) multifon.Routing {
	for k, v := range multifon.RoutingDescriptionMap {
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
		args[i] = "{" + k + "}"
		args[i+1] = fmt.Sprint(v)
		i += 2
	}
	return strings.NewReplacer(args...).Replace(s)
}

func printUsage() {
	var name string
	if len(os.Args) == 0 {
		name = "PROG_NAME"
	} else {
		name = filepath.Base(os.Args[0])
	}
	apiChoices := getAPIChoices()
	sort.Strings(apiChoices)
	routingDescriptions := getRoutingDescriptions()
	sort.Strings(routingDescriptions)
	sep := " | "
	formatMap := map[string]interface{}{
		"NAME":   name,
		"TAB":    strings.Repeat(" ", len(name)),
		"CMD":    MetaVarCommand,
		"CMDARG": MetaVarCommandArg,
		"STDIN":  ArgStdin,
		"DEFA":   multifon.DefaultAPI,
		"DEFT":   multifon.DefaultTimeout,
		"ENVL":   EnvLogin,
		"ENVP":   EnvPassword,
		"ENVA":   EnvAPI,
		"ENVT":   EnvTimeout,
		"ENVNP":  EnvNewPassword,
		"CA":     strings.Join(apiChoices, sep),
		"CCMD":   strings.Join(Commands[:], sep),
		"CCMDR":  strings.Join(routingDescriptions, sep),
		"CMDR":   CommandRouting,
		"CMDL":   CommandLines,
		"CMDSP":  CommandSetPassword,
	}
	for _, v := range []Flag{FlagHelp, FlagVersion} {
		formatMap[v.ShortName] = v.ShortName
	}
	for _, v := range []Flag{
		FlagConfig,
		FlagLogin,
		FlagPassword,
		FlagAPI,
		FlagTimeout,
	} {
		formatMap[v.ShortName] = v.ShortName
		formatMap[v.ShortName+v.ShortName] = v.LongName
		formatMap[strings.ToUpper(v.ShortName)] = v.MetaVar()
	}
	fmt.Fprintln(
		flag.CommandLine.Output(),
		format(`{NAME} [-{h}{V}] ( -{cc} <{C}> | -{ll} <{L}> -{pp} <{P}> )
{TAB} [-{aa} <{A}>] [-{tt} <{T}>] <{CMD}> [<{CMDARG}>]

[-{h}] * Print help and exit
[-{V}] * Print version and exit

-{c}, -{cc}:
  filepath (stdin: {STDIN})

-{l}, -{ll}:
  string (env: {ENVL})

-{p}, -{pp}:
  string (env: {ENVP})

-{a}, -{aa}:
  { {CA} } (default: {DEFA}, env: {ENVA})

-{t}, -{tt}:
  time.ParseDuration (default: {DEFT}, env: {ENVT})

{CMD}:
  { {CCMD} }

{CMDARG}:
  {CMDR} { {CCMDR} }
  {CMDL} <NUMBER> (2 .. 20)
  {CMDSP} <NEW_{P}>
      tip: 8 <= x <= 20, mixed case, digits
      env: {ENVNP}`,
			formatMap,
		),
	)
}

func fatalParseArg(k, v string) {
	fmt.Fprintf(os.Stderr, "Failed to parse argument <%s>: `%s`\n", k, v)
	os.Exit(ExitUsage)
}

func parseCommand(opts *Opts) {
	cmd := strings.ToLower(flag.Arg(0))
	for _, v := range Commands {
		if v == cmd {
			opts.command = cmd
			return
		}
	}
	fatalParseArg(MetaVarCommand, cmd)
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
			fatalParseArg(MetaVarCommandArg, arg)
		}
		opts.commandArg = routing
	case CommandLines:
		if arg == "" {
			return
		}
		n, err := strconv.Atoi(arg)
		if err != nil {
			fatalParseArg(MetaVarCommandArg, arg)
		}
		opts.commandArg = n
	case CommandSetPassword:
		if !parseIdentity(&arg, opts.config.NewPassword, EnvNewPassword) {
			fatalParseArg(MetaVarCommandArg, arg)
		}
		opts.commandArg = arg
	}
}

func parseIdentity(arg *string, configValue, envKey string) bool {
	if *arg == "" {
		val := configValue
		if val == "" {
			val = os.Getenv(envKey)
			if val == "" {
				return false
			}
		}
		*arg = val
	}
	return true
}

func parseAPI(opts *Opts) {
	if opts.api == "" {
		if val := os.Getenv(EnvAPI); val != "" {
			err := opts.api.Set(val)
			if err == nil {
				return
			}
			log.Println(err)
		}
		opts.api = opts.config.API
	}
}

func parseTimeout(opts *Opts) {
	if opts.timeout < 0 {
		if val := os.Getenv(EnvTimeout); val != "" {
			timeout, err := time.ParseDuration(val)
			if err == nil {
				opts.timeout = timeout
				return
			}
			log.Println(err)
		}
		if opts.config.Timeout != nil {
			opts.timeout = opts.config.Timeout.Unwrap()
		}
	}
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
		os.Exit(ExitUsage)
	}
	var isHelp bool
	var isVersion bool
	opts := &Opts{}
	for _, s := range FlagHelp.Names() {
		flag.BoolVar(&isHelp, s, false, "")
	}
	for _, s := range FlagVersion.Names() {
		flag.BoolVar(&isVersion, s, false, "")
	}
	for _, s := range FlagConfig.Names() {
		flag.Var(&opts.config, s, "")
	}
	for _, s := range FlagLogin.Names() {
		flag.StringVar(&opts.login, s, "", "")
	}
	for _, s := range FlagPassword.Names() {
		flag.StringVar(&opts.password, s, "", "")
	}
	for _, s := range FlagAPI.Names() {
		flag.Var(&opts.api, s, "")
	}
	for _, s := range FlagTimeout.Names() {
		flag.DurationVar(&opts.timeout, s, -1, "")
	}
	flag.Parse()
	if isHelp {
		flag.Usage()
		os.Exit(ExitSuccess)
	}
	if isVersion {
		fmt.Println(multifon.Version)
		os.Exit(ExitSuccess)
	}
	if !parseIdentity(&opts.login, opts.config.Login, EnvLogin) {
		fatalParseArg(FlagLogin.MetaVar(), opts.login)
	}
	if !parseIdentity(&opts.password, opts.config.Password, EnvPassword) {
		fatalParseArg(FlagPassword.MetaVar(), opts.password)
	}
	parseCommand(opts)
	parseCommandArg(opts)
	parseAPI(opts)
	parseTimeout(opts)
	return opts
}

func updateConfigFile(opts *Opts) error {
	file, err := os.OpenFile(
		opts.config.path,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0600,
	)
	if err != nil {
		return err
	}
	opts.config.Password = opts.commandArg.(string)
	opts.config.NewPassword = opts.password
	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")
	err = enc.Encode(opts.config)
	if err1 := file.Close(); err == nil {
		err = err1
	}
	return err
}

func main() {
	opts := parseArgs()
	var httpClient *http.Client
	if opts.timeout >= 0 {
		httpClient = &http.Client{Timeout: opts.timeout}
	}
	client := multifon.NewClient(
		opts.login,
		opts.password,
		opts.api.Unwrap(),
		httpClient,
	)
	fatalIfErr := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(ExitFailure)
		}
	}
	ctx := context.Background()
	strOk := "OK"
	switch opts.command {
	case CommandBalance:
		res, err := client.GetBalance(ctx)
		fatalIfErr(err)
		fmt.Println(res.Balance)
	case CommandRouting:
		if opts.commandArg == nil {
			res, err := client.GetRouting(ctx)
			fatalIfErr(err)
			val := res.Description()
			if val == "" {
				fmt.Println(res.Routing)
			} else {
				fmt.Println(val)
			}
		} else {
			_, err := client.SetRouting(
				ctx,
				opts.commandArg.(multifon.Routing),
			)
			fatalIfErr(err)
			fmt.Println(strOk)
		}
	case CommandStatus:
		res, err := client.GetStatus(ctx)
		fatalIfErr(err)
		val := res.Description()
		if val == "" {
			val = strconv.Itoa(res.Status)
		}
		if res.Expires != "" {
			val += ":" + res.Expires
		}
		fmt.Println(val)
	case CommandProfile:
		res, err := client.GetProfile(ctx)
		fatalIfErr(err)
		fmt.Println(res.MSISDN)
	case CommandLines:
		if opts.commandArg == nil {
			res, err := client.GetLines(ctx)
			fatalIfErr(err)
			fmt.Println(res.Lines)
		} else {
			_, err := client.SetLines(ctx, opts.commandArg.(int))
			fatalIfErr(err)
			fmt.Println(strOk)
		}
	case CommandSetPassword:
		_, err := client.SetPassword(ctx, opts.commandArg.(string))
		fatalIfErr(err)
		if opts.config.path != "" {
			fatalIfErr(updateConfigFile(opts))
		}
		fmt.Println(strOk)
	}
}
