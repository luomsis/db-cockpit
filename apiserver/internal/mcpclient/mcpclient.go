// Package mcpclient 提供插件域（D15）所需的最小 MCP http 客户端：
// JSON-RPC 2.0 over HTTP POST，仅服务 apiserver 侧的触发式发现（initialize + tools/list）
// 与连通性健康检查。工具调用本身由 agentcluster 直连 MCP server 完成，不经本包。
package mcpclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Tool MCP tools/list 返回的工具条目（MVP 只取发现所需字段）。
type Tool struct {
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type rpcRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

const protocolVersion = "2025-03-26"

// Initialize MCP 握手。宽松策略：部分无状态 http server 不要求握手，
// 调用方对失败可忽略、以 tools/list 结果为准。
func Initialize(ctx context.Context, baseURL string) error {
	params := map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]string{"name": "db-cockpit-apiserver", "version": "0.1.0"},
	}
	_, err := call(ctx, baseURL, "initialize", params)
	return err
}

// ToolsList 拉取工具清单（含 inputSchema，发现草案的数据来源）。
func ToolsList(ctx context.Context, baseURL string) ([]Tool, error) {
	raw, err := call(ctx, baseURL, "tools/list", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse tools/list result: %w", err)
	}
	return result.Tools, nil
}

func call(ctx context.Context, baseURL, method string, params map[string]interface{}) (json.RawMessage, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("empty base url")
	}
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mcp server http %d", resp.StatusCode)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, err
	}
	var rpc rpcResponse
	if err := json.Unmarshal(payload, &rpc); err != nil {
		return nil, fmt.Errorf("parse rpc response: %w", err)
	}
	if rpc.Error != nil {
		return nil, fmt.Errorf("mcp rpc error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}
	return rpc.Result, nil
}

// decodeResponse 兼容两种响应承载：application/json（直发 JSON-RPC）与
// text/event-stream（streamable-http，取首个 data: 帧的 JSON-RPC 响应）。
func decodeResponse(resp *http.Response) ([]byte, error) {
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 4<<20))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if after, ok := strings.CutPrefix(line, "data:"); ok {
			after = strings.TrimSpace(after)
			if after != "" && after != "[DONE]" {
				return []byte(after), nil
			}
		}
	}
	return nil, fmt.Errorf("no data frame in event-stream response")
}
