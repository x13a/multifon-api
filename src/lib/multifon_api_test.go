package multifonapi

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	TestConfPath = filepath.Join("testdata", "conf.json")
	Config       TestConf
)

type DescriptionResponse interface {
	Description() string
}

type TestConf struct {
	Login, Password string
	NewPassword     string `json:"new_password"`
}

func loadTestConf() error {
	file, err := os.Open(TestConfPath)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&Config); err != nil {
		return err
	}
	return nil
}

func call(obj interface{}, name string, args ...interface{}) []reflect.Value {
	arguments := make([]reflect.Value, len(args))
	for i, v := range args {
		arguments[i] = reflect.ValueOf(v)
	}
	return reflect.ValueOf(obj).MethodByName(name).Call(arguments)
}

func get(t *testing.T, fnName string) {
	for k := range APIUrlMap {
		t.Run(k.String(), func(t *testing.T) {
			c := NewClient(Config.Login, Config.Password, k, nil)
			res := call(c, fnName)
			if err, ok := res[1].Interface().(error); ok && err != nil {
				t.Fatal(err)
			}
			if obj, ok := res[0].Interface().(DescriptionResponse); ok &&
				obj.Description() == "" {

				t.Fatalf("empty response description, %+v", obj)
			}
		})
	}
}

func set(t *testing.T, fnName string, values []interface{}) {
	setPrefix := "Set"
	getFnName := strings.Replace(fnName, setPrefix, "Get", 1)
	name := fnName[len(setPrefix):]
	_set := func(t *testing.T, c *Client, v interface{}) bool {
		res := call(c, fnName, v)
		if err, ok := res[1].Interface().(error); ok && err != nil {
			t.Error(err)
			return false
		}
		return true
	}
	for k := range APIUrlMap {
		t.Run(k.String(), func(t *testing.T) {
			c := NewClient(Config.Login, Config.Password, k, nil)
			res := call(c, getFnName)
			if err, ok := res[1].Interface().(error); ok && err != nil {
				t.Fatal(err)
			}
			initVal := res[0].Elem().FieldByName(name).Interface()
			for _, val := range values {
				if val == initVal {
					continue
				}
				_set(t, c, val)
				time.Sleep(1 * time.Second)
			}
			_set(t, c, initVal)
		})
	}
}

func TestMain(m *testing.M) {
	if err := loadTestConf(); err != nil {
		log.Fatal(err)
	}
	if Config.Login == "" {
		log.Fatal("login required")
	}
	if Config.Password == "" {
		log.Fatal("password required")
	}
	os.Exit(m.Run())
}

func TestGetBalance(t *testing.T) {
	get(t, "GetBalance")
}

func TestGetRouting(t *testing.T) {
	get(t, "GetRouting")
}

func TestSetRouting(t *testing.T) {
	set(t, "SetRouting", []interface{}{RoutingGSM, RoutingSIP, RoutingSIPGSM})
}

func TestGetStatus(t *testing.T) {
	get(t, "GetStatus")
}

func TestGetProfile(t *testing.T) {
	get(t, "GetProfile")
}

func TestGetLines(t *testing.T) {
	get(t, "GetLines")
}

func TestSetLines(t *testing.T) {
	set(t, "SetLines", []interface{}{2, 3})
}

func TestSetPassword(t *testing.T) {
	if Config.NewPassword == "" {
		t.Fatal("new_password required")
	}
	for k := range APIUrlMap {
		t.Run(k.String(), func(t *testing.T) {
			c := NewClient(Config.Login, Config.Password, k, nil)
			for _, passwd := range [...]string{
				Config.NewPassword,
				Config.Password,
			} {
				if _, err := c.SetPassword(passwd); err != nil {
					t.Fatal(err)
				}
				time.Sleep(1 * time.Second)
			}
		})
	}
}
