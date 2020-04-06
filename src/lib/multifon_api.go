package multifonapi

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type (
	API     string
	Routing int
)

func (a API) String() string {
	return string(a)
}

const (
	Version = "0.1.1"

	APIMultifon API = "multifon"
	APIEmotion  API = "emotion"
	APIDefault      = APIMultifon

	RoutingGSM    Routing = 0
	RoutingSIP    Routing = 1
	RoutingSIPGSM Routing = 2

	StatusActive  = 0
	StatusBlocked = 1

	DefaultTimeout = 30 * time.Second
)

var (
	APIUrlMap = map[API]string{
		APIMultifon: "https://sm.megafon.ru/sm/client/",
		APIEmotion:  "https://emotion.megalabs.ru/sm/client/",
	}
	RoutingDescriptionMap = map[Routing]string{
		RoutingGSM:    "GSM",
		RoutingSIP:    "SIP",
		RoutingSIPGSM: "SIP+GSM",
	}
)

type ResultError struct {
	Code        int
	Description string
}

func (e *ResultError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Description)
}

type HTTPStatusError struct {
	Code   int
	Status string
}

func (e *HTTPStatusError) Error() string {
	return e.Status
}

type SetFailedError struct {
	Key          string
	Value        interface{}
	CurrentValue interface{}
}

func (e *SetFailedError) Error() string {
	return fmt.Sprintf(
		"Failed to set %s, invalid value: %v (current: %v)",
		e.Key,
		e.Value,
		e.CurrentValue,
	)
}

type Response interface {
	ResultError() error
}

type ResponseResult struct {
	XMLName xml.Name `xml:"response"`
	Result  struct {
		Code        int    `xml:"code"`
		Description string `xml:"description"`
	} `xml:"result"`
}

func (r *ResponseResult) ResultError() error {
	if r.Result.Code != http.StatusOK {
		return &ResultError{
			Code:        r.Result.Code,
			Description: r.Result.Description,
		}
	}
	return nil
}

type ResponseBalance struct {
	ResponseResult
	Balance float64 `xml:"balance"`
}

type ResponseRouting struct {
	ResponseResult
	Routing Routing `xml:"routing"`
}

func (r *ResponseRouting) Description() string {
	if v, ok := RoutingDescriptionMap[r.Routing]; ok {
		return v
	}
	return ""
}

type ResponseStatus struct {
	ResponseResult
	Status  int    `xml:"status"`
	Expires string `xml:"expires"` // [U]Int?
}

func (r *ResponseStatus) Description() string {
	switch r.Status {
	case StatusActive:
		return "active"
	case StatusBlocked:
		return "blocked"
	default:
		return ""
	}
}

type ResponseProfile struct {
	ResponseResult
	MSISDN string `xml:"msisdn"`
}

type ResponseLines struct {
	ResponseResult
	Lines int `xml:"ParallelCallsSipOut"`
}

type Client struct {
	login      string
	password   string
	apiUrl     string
	httpClient *http.Client
}

func (c *Client) GetLogin() string {
	return c.login
}

func (c *Client) SetAPI(api API) {
	c.apiUrl = APIUrlMap[api]
}

func (c *Client) Request(
	urlPath string,
	params map[string]string,
) (*http.Request, error) {
	reqUrl, err := urlJoin(c.apiUrl, urlPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, reqUrl, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("login", c.login)
	q.Add("password", c.password)
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}

func (c *Client) Do(
	urlPath string,
	params map[string]string,
	data Response,
) error {
	req, err := c.Request(urlPath, params)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode < 600 {
		return &HTTPStatusError{
			Code:   resp.StatusCode,
			Status: resp.Status,
		}
	}
	if err := xml.NewDecoder(resp.Body).Decode(data); err != nil {
		return err
	}
	return data.ResultError()
}

func (c *Client) GetBalance() (*ResponseBalance, error) {
	data := &ResponseBalance{}
	return data, c.Do("balance", nil, data)
}

func (c *Client) GetRouting() (*ResponseRouting, error) {
	data := &ResponseRouting{}
	return data, c.Do("routing", nil, data)
}

/*
RoutingGSM, RoutingSIP, RoutingSIPGSM
*/
func (c *Client) SetRouting(routing Routing) (*ResponseRouting, error) {
	k := "routing"
	v := Routing(-1)
	data := &ResponseRouting{Routing: v}
	err := c.Do(k, map[string]string{k: strconv.Itoa(int(routing))}, data)
	if err == nil && data.Routing != v {
		err = &SetFailedError{
			Key:          k,
			Value:        routing,
			CurrentValue: data.Routing,
		}
	}
	return data, err
}

func (c *Client) GetStatus() (*ResponseStatus, error) {
	data := &ResponseStatus{}
	return data, c.Do("status", nil, data)
}

/*
outdated?
*/
func (c *Client) GetProfile() (*ResponseProfile, error) {
	data := &ResponseProfile{}
	return data, c.Do("profile", nil, data)
}

func (c *Client) GetLines() (*ResponseLines, error) {
	data := &ResponseLines{}
	return data, c.Do("lines", nil, data)
}

/*
2 .. 20
*/
func (c *Client) SetLines(n int) (*ResponseLines, error) {
	k := "lines"
	v := -1
	data := &ResponseLines{Lines: v}
	err := c.Do(k, map[string]string{k: strconv.Itoa(n)}, data)
	if err == nil && data.Lines != v {
		err = &SetFailedError{
			Key:          k,
			Value:        n,
			CurrentValue: data.Lines,
		}
	}
	return data, err
}

/*
min 8, max 20, mixed case, digits
can return error and.. change password on server?
*/
func (c *Client) SetPassword(password string) (*ResponseResult, error) {
	data := &ResponseResult{}
	err := c.Do("password", map[string]string{"new_password": password}, data)
	if err == nil {
		c.password = password
	}
	return data, err
}

func NewClient(
	login string,
	password string,
	api API,
	httpClient *http.Client,
) *Client {
	if api == "" {
		api = APIDefault
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	return &Client{
		login:      login,
		password:   password,
		apiUrl:     APIUrlMap[api],
		httpClient: httpClient,
	}
}
