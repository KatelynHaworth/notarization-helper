package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	idFormat = "20060102150405-.999"
)

type rpcResponse struct {
	Result json.RawMessage `json:"Result"`
}

func generateId() string {
	id := time.Now().Format(idFormat)

	return strings.ReplaceAll(id, ".", "")
}

func (*Worker) jsonRpcSendRequest(url, method string, params, response interface{}) error {
	request := resty.New().NewRequest()
	request.SetResult(new(rpcResponse))
	request.SetHeaders(map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	request.SetBody(struct {
		Id      string      `json:"id"`
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
	}{
		Id:      generateId(),
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})

	resp, err := request.Post(url)
	switch {
	case err == nil && resp.StatusCode() != http.StatusOK:
		err = fmt.Errorf("unexpected response code: %s", resp.Status())
		fallthrough

	case err != nil:
		return fmt.Errorf("send JSON-RPC request: %w", err)

	default:
		rpcResp := resp.Result().(*rpcResponse)
		if err = json.Unmarshal(rpcResp.Result, response); err != nil {
			return fmt.Errorf("unmarshal JSON-RPC result: %w", err)
		}

		return nil
	}
}
