package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"

	"whatsd/internal/types"
)

type Client struct {
	SocketPath string
}

func NewClient(socketPath string) *Client {
	return &Client{SocketPath: socketPath}
}

func (c *Client) Dial() (net.Conn, error) {
	conn, err := net.Dial("unix", c.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("could not connect to whatsd socket at %s: %w", c.SocketPath, err)
	}
	return conn, nil
}

func (c *Client) Call(method string, params map[string]any) (*types.Response, error) {
	conn, err := c.Dial()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: method,
		Params: params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	data = append(data, '\n')

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to socket: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var resp types.Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("daemon error: %s", resp.Error)
		}
		return &resp, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	return nil, fmt.Errorf("empty response from daemon")
}

func (c *Client) CallAndPrint(method string, params map[string]any) error {
	resp, err := c.Call(method, params)
	if err != nil {
		return err
	}

	if resp.Result != nil {
		resultBytes, err := json.MarshalIndent(resp.Result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
		fmt.Println(string(resultBytes))
	}
	return nil
}
