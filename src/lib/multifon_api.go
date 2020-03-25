package multifonapi

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	API_MULTIFON = "multifon"
	API_EMOTION  = "emotion"
	API_DEFAULT  = API_MULTIFON

	ROUTING_GSM     = 0
	ROUTING_SIP     = 1
	ROUTING_SIP_GSM = 2

	STATUS_ACTIVE  = 0
	STATUS_BLOCKED = 1

	DEFAULT_TIMEOUT = 30 * time.Second
)

var (
	API_NAME_URL_MAP = map[string]string{
		API_MULTIFON: "https://sm.megafon.ru/sm/client/",
		API_EMOTION:  "https://emotion.megalabs.ru/sm/client/",
	}
	ROUTING_DESCRIPTION_MAP = map[int]string{
		ROUTING_GSM:     "GSM",
		ROUTING_SIP:     "SIP",
		ROUTING_SIP_GSM: "SIP+GSM",
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
	Routing int `xml:"routing"`
}

func (r *ResponseRouting) Description() string {
	if v, ok := ROUTING_DESCRIPTION_MAP[r.Routing]; ok {
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
	case STATUS_ACTIVE:
		return "active"
	case STATUS_BLOCKED:
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

func newSetFailedError(key string, value, currentValue interface{}) error {
	return fmt.Errorf(
		"Failed to set %s, invalid value: %v (current: %v)",
		key,
		value,
		currentValue,
	)
}

type Client struct {
	login      string
	password   string
	apiUrl     string
	httpClient *http.Client
}

func (c *Client) Request(
	urlPath string,
	params map[string]string,
) ([]byte, error) {
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
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode < 600 {
		return nil, &HTTPStatusError{
			Code:   resp.StatusCode,
			Status: resp.Status,
		}
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *Client) Do(
	urlPath string,
	params map[string]string,
	data Response,
) error {
	buf, err := c.Request(urlPath, params)
	if err != nil {
		return err
	}
	if err := xml.Unmarshal(buf, data); err != nil {
		return err
	}
	if err := data.ResultError(); err != nil {
		return err
	}
	return nil
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
ROUTING_GSM, ROUTING_SIP, ROUTING_SIP_GSM
*/
func (c *Client) SetRouting(routing int) (*ResponseRouting, error) {
	k := "routing"
	v := -1
	data := &ResponseRouting{Routing: v}
	err := c.Do(k, map[string]string{k: strconv.Itoa(routing)}, data)
	if err == nil && data.Routing != v {
		err = newSetFailedError(k, routing, data.Routing)
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
		err = newSetFailedError(k, n, data.Lines)
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
	login, password, api string,
	httpClient *http.Client,
) *Client {
	apiUrl, ok := API_NAME_URL_MAP[api]
	if !ok {
		apiUrl = API_NAME_URL_MAP[API_DEFAULT]
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DEFAULT_TIMEOUT}
	}
	return &Client{
		login:      login,
		password:   password,
		apiUrl:     apiUrl,
		httpClient: httpClient,
	}
}
