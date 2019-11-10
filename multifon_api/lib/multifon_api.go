package multifonapi

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	MULTIFON_API_URL = "https://sm.megafon.ru/sm/client/"
	EMOTION_API_URL  = "https://emotion.megalabs.ru/sm/client/"
	DEFAULT_API_URL  = MULTIFON_API_URL

	ROUTING_GSM     = 0
	ROUTING_SIP     = 1
	ROUTING_SIP_GSM = 2

	STATUS_ACTIVE  = 0
	STATUS_BLOCKED = 1
)

var ROUTING_DESCRIPTION_MAP = map[int]string{
	ROUTING_GSM:     "GSM",
	ROUTING_SIP:     "SIP",
	ROUTING_SIP_GSM: "SIP+GSM",
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

func (s *ResponseResult) ResultError() error {
	if s.Result.Code != http.StatusOK {
		return errors.New(s.Result.Description)
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

func (s *ResponseRouting) Description() string {
	if v, ok := ROUTING_DESCRIPTION_MAP[s.Routing]; ok {
		return v
	}
	return ""
}

type ResponseStatus struct {
	ResponseResult
	Status  int    `xml:"status"`
	Expires string `xml:"expires"` // [U]Int?
}

func (s *ResponseStatus) Description() string {
	switch s.Status {
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
	httpClient *http.Client
	login      string
	password   string
	apiUrl     string
}

func (s *Client) Request(
	urlPath string,
	params map[string]string,
) ([]byte, error) {
	reqUrl, err := urlJoin(s.apiUrl, urlPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, reqUrl, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("login", s.login)
	q.Add("password", s.password)
	if params != nil {
		for k, v := range params {
			q.Add(k, v)
		}
	}
	req.URL.RawQuery = q.Encode()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode < 600 {
		return nil, errors.New(resp.Status)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *Client) Do(
	urlPath string,
	params map[string]string,
	data Response,
) error {
	buf, err := s.Request(urlPath, params)
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

func (s *Client) GetBalance() (*ResponseBalance, error) {
	data := &ResponseBalance{}
	return data, s.Do("balance", nil, data)
}

func (s *Client) GetRouting() (*ResponseRouting, error) {
	data := &ResponseRouting{}
	return data, s.Do("routing", nil, data)
}

/*
ROUTING_GSM, ROUTING_SIP, ROUTING_SIP_GSM
*/
func (s *Client) SetRouting(routing int) (*ResponseRouting, error) {
	k := "routing"
	v := -1
	data := &ResponseRouting{Routing: v}
	err := s.Do(k, map[string]string{k: strconv.Itoa(routing)}, data)
	if err == nil && data.Routing != v {
		err = newSetFailedError(k, routing, data.Routing)
	}
	return data, err
}

func (s *Client) GetStatus() (*ResponseStatus, error) {
	data := &ResponseStatus{}
	return data, s.Do("status", nil, data)
}

/*
outdated?
*/
func (s *Client) GetProfile() (*ResponseProfile, error) {
	data := &ResponseProfile{}
	return data, s.Do("profile", nil, data)
}

func (s *Client) GetLines() (*ResponseLines, error) {
	data := &ResponseLines{}
	return data, s.Do("lines", nil, data)
}

/*
2 .. 20
*/
func (s *Client) SetLines(n int) (*ResponseLines, error) {
	k := "lines"
	v := -1
	data := &ResponseLines{Lines: v}
	err := s.Do(k, map[string]string{k: strconv.Itoa(n)}, data)
	if err == nil && data.Lines != v {
		err = newSetFailedError(k, n, data.Lines)
	}
	return data, err
}

/*
min 8, max 20, mixed case, digits
can return error and.. change password on server?
*/
func (s *Client) SetPassword(password string) (*ResponseResult, error) {
	data := &ResponseResult{}
	err := s.Do("password", map[string]string{"new_password": password}, data)
	if err == nil {
		s.password = password
	}
	return data, err
}

func NewClient(
	login, password, apiUrl string,
	httpClient *http.Client,
) *Client {
	if apiUrl == "" {
		apiUrl = DEFAULT_API_URL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		httpClient: httpClient,
		login:      login,
		password:   password,
		apiUrl:     apiUrl,
	}
}
