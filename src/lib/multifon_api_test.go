package multifonapi

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const (
	FlagConfig  = "config"
	FlagSkipass = "skipass"
)

var (
	ConfigPath = filepath.Join("testdata", "conf.json")
	Config     ConfigType

	Skipass bool
)

type DescriptionResponse interface {
	Description() string
}

type ConfigType struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

func parseFlag() {
	flag.StringVar(
		&ConfigPath,
		FlagConfig,
		ConfigPath,
		"Path to configuration file",
	)
	flag.BoolVar(
		&Skipass,
		FlagSkipass,
		false,
		"Skip `SetPassword` test",
	)
	flag.Parse()
}

func loadConfig() error {
	file, err := os.Open(ConfigPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(&Config)
}

func call(obj interface{}, name string, args ...interface{}) []reflect.Value {
	arguments := make([]reflect.Value, len(args))
	for i, v := range args {
		arguments[i] = reflect.ValueOf(v)
	}
	return reflect.ValueOf(obj).MethodByName(name).Call(arguments)
}

func reflectError(v []reflect.Value) error {
	err, _ := v[len(v)-1].Interface().(error)
	return err
}

func getCall(t *testing.T, c *Client, name string) reflect.Value {
	v := call(c, name, context.Background())
	if err := reflectError(v); err != nil {
		t.Fatal(err)
	}
	return v[0]
}

func getFnName(s string) string {
	return "Get" + s
}

func newClient(api API) *Client {
	return NewClient(Config.Login, Config.Password, api, nil)
}

func delay() {
	time.Sleep(1 * time.Second)
}

func get(t *testing.T, name string) {
	fnName := getFnName(name)
	for api := range APIUrlMap {
		api := api
		t.Run(api.String(), func(t *testing.T) {
			if obj, ok := getCall(t, newClient(api), fnName).
				Interface().(DescriptionResponse); ok &&
				obj.Description() == "" {

				t.Fatalf("empty response description, %+v", obj)
			}
		})
	}
}

func set(t *testing.T, name string, values []interface{}) {
	getFnName := getFnName(name)
	setFnName := "Set" + name
	_set := func(t *testing.T, c *Client, v interface{}) {
		if err := reflectError(call(
			c,
			setFnName,
			context.Background(),
			v,
		)); err != nil {
			t.Error(err)
		}
	}
	for api := range APIUrlMap {
		api := api
		t.Run(api.String(), func(t *testing.T) {
			c := newClient(api)
			initVal := getCall(t, c, getFnName).
				Elem().
				FieldByName(name).
				Interface()
			delay()
			for _, val := range values {
				if val == initVal {
					continue
				}
				_set(t, c, val)
				delay()
			}
			_set(t, c, initVal)
		})
	}
}

func TestMain(m *testing.M) {
	parseFlag()
	if err := loadConfig(); err != nil {
		log.Fatalln(err)
	}
	if Config.Login == "" {
		log.Fatalln("login required")
	}
	if Config.Password == "" {
		log.Fatalln("password required")
	}
	os.Exit(m.Run())
}

func TestGetBalance(t *testing.T) {
	get(t, "Balance")
}

func TestGetRouting(t *testing.T) {
	get(t, "Routing")
}

func TestSetRouting(t *testing.T) {
	set(t, "Routing", []interface{}{RoutingGSM, RoutingSIP, RoutingSIPGSM})
}

func TestGetStatus(t *testing.T) {
	get(t, "Status")
}

func TestGetProfile(t *testing.T) {
	get(t, "Profile")
}

func TestGetLines(t *testing.T) {
	get(t, "Lines")
}

func TestSetLines(t *testing.T) {
	set(t, "Lines", []interface{}{2, 3})
}

func TestSetPassword(t *testing.T) {
	if Skipass {
		t.SkipNow()
	}
	if Config.NewPassword == "" {
		t.Fatal("new_password required")
	}
	for api := range APIUrlMap {
		api := api
		t.Run(api.String(), func(t *testing.T) {
			c := newClient(api)
			for _, password := range [...]string{
				Config.NewPassword,
				Config.Password,
			} {
				if _, err := c.SetPassword(
					context.Background(),
					password,
				); err != nil {
					t.Fatal(err)
				}
				delay()
			}
		})
	}
}
