package multifonapi

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	Version = "0.1.13"

	APIMultifon API = "multifon"
	APIEmotion  API = "emotion"

	RoutingGSM    Routing = 0
	RoutingSIP    Routing = 1
	RoutingSIPGSM Routing = 2

	StatusActive  = 0
	StatusBlocked = 1

	DefaultAPI     = APIMultifon
	DefaultTimeout = 1 << 5 * time.Second
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

type (
	API     string
	Routing int
)

func (a API) String() string {
	return string(a)
}

type ResultError struct {
	Code        int
	Description string
}

func (e *ResultError) Error() string {
	return strconv.Itoa(e.Code) + ": " + e.Description
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
		return &ResultError{r.Result.Code, r.Result.Description}
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
	apiURL     *url.URL
	httpClient *http.Client
}

func (c *Client) GetLogin() string {
	return c.login
}

func (c *Client) SetAPI(api API) {
	u, err := url.ParseRequestURI(APIUrlMap[api])
	if err != nil {
		panic(err)
	}
	if u.Scheme != "https" {
		panic("insecure scheme")
	}
	c.apiURL = u
}

func (c *Client) request(
	ctx context.Context,
	apiReference string,
	params map[string]string,
) (*http.Request, error) {
	ref, err := url.Parse(apiReference)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.apiURL.ResolveReference(ref).String(),
		nil,
	)
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
	ctx context.Context,
	apiReference string,
	params map[string]string,
	data Response,
) error {
	req, err := c.request(ctx, apiReference, params)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode < 600 {
		return &HTTPStatusError{resp.StatusCode, resp.Status}
	}
	if err = xml.NewDecoder(resp.Body).Decode(data); err != nil {
		return err
	}
	return data.ResultError()
}

func (c *Client) GetBalance(ctx context.Context) (*ResponseBalance, error) {
	data := &ResponseBalance{}
	return data, c.Do(ctx, "balance", nil, data)
}

func (c *Client) GetRouting(ctx context.Context) (*ResponseRouting, error) {
	data := &ResponseRouting{}
	return data, c.Do(ctx, "routing", nil, data)
}

/*
RoutingGSM, RoutingSIP, RoutingSIPGSM
*/
func (c *Client) SetRouting(
	ctx context.Context,
	routing Routing,
) (*ResponseRouting, error) {
	k := "routing"
	v := Routing(-1)
	data := &ResponseRouting{Routing: v}
	err := c.Do(ctx, k, map[string]string{k: strconv.Itoa(int(routing))}, data)
	if err == nil && data.Routing != v {
		err = &SetFailedError{k, routing, data.Routing}
	}
	return data, err
}

func (c *Client) GetStatus(ctx context.Context) (*ResponseStatus, error) {
	data := &ResponseStatus{}
	return data, c.Do(ctx, "status", nil, data)
}

/*
outdated?
*/
func (c *Client) GetProfile(ctx context.Context) (*ResponseProfile, error) {
	data := &ResponseProfile{}
	return data, c.Do(ctx, "profile", nil, data)
}

func (c *Client) GetLines(ctx context.Context) (*ResponseLines, error) {
	data := &ResponseLines{}
	return data, c.Do(ctx, "lines", nil, data)
}

/*
2 .. 20
*/
func (c *Client) SetLines(ctx context.Context, n int) (*ResponseLines, error) {
	k := "lines"
	v := -1
	data := &ResponseLines{Lines: v}
	err := c.Do(ctx, k, map[string]string{k: strconv.Itoa(n)}, data)
	if err == nil && data.Lines != v {
		err = &SetFailedError{k, n, data.Lines}
	}
	return data, err
}

/*
min 8, max 20, mixed case, digits
can return error and.. change password on server?
*/
func (c *Client) SetPassword(
	ctx context.Context,
	password string,
) (*ResponseResult, error) {
	data := &ResponseResult{}
	err := c.Do(
		ctx,
		"password",
		map[string]string{"new_password": password},
		data,
	)
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
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	c := &Client{
		login:      login,
		password:   password,
		httpClient: httpClient,
	}
	if api == "" {
		api = DefaultAPI
	}
	c.SetAPI(api)
	return c
}
