package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	httpclient *http.Client
	origin     string
	username   string
	password   string
}

func NewClient(origin, username, password string) *Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	return &Client{
		httpclient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second, // Overall request timeout
		},
		origin:   origin,
		username: username,
		password: password,
	}
}

func (client *Client) do(ctx context.Context, method string, path string, body any, target any) error {
	var reader io.Reader = nil
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewBuffer(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, client.origin+"/"+path, reader)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return err
	}
	req.SetBasicAuth(client.username, client.password)
	res, err := client.httpclient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("HTTP %d: %s", res.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(res.Body).Decode(target)
}

func (client *Client) rest(ctx context.Context, method string, path []string, body any, target any) error {
	p := strings.Join(path, "/")
	return client.do(ctx, method, p, body, target)
}

type rpcBody struct {
	Method  string `json:"method"`
	Params  []any  `json:"params,omitempty"`
	JsonRPC string `json:"jsonrpc"`
	Id      int    `json:"id"`
}

type RpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	ID      int             `json:"id"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (client *Client) Rpc(ctx context.Context, method string, params []any, target any) error {
	body := rpcBody{method, params, "2.0", 1337}
	response := RpcResponse{}
	if err := client.do(ctx, "POST", "", &body, &response); err != nil {
		return err
	}
	if response.Error != nil {
		return fmt.Errorf("rpc client: %v", response.Error.Message)
	}

	return json.Unmarshal(response.Result, target)
}
