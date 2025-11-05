// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcpproxy

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewMCPProxyCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "mcp-proxy",
		Short: i18n.T("Start MCP server proxy", "启动 MCP 服务器代理"),
		Long: i18n.T(
			"Start a local proxy server for Aliyun API MCP Servers. "+
				"The proxy handles OAuth authentication automatically, "+
				"allowing MCP clients to connect without managing credentials.",
			"启动阿里云 API MCP Server 的本地代理服务。"+
				"代理自动处理 OAuth 认证，"+
				"允许 MCP 客户端无需管理凭证即可连接。",
		),
		Usage:  "aliyun mcp-proxy [--mcp-profile PROFILE] [--port PORT] [--host HOST] [--region REGION] [--bearer-token TOKEN]",
		Sample: "aliyun mcp-proxy --region CN --port 8088",
		Run: func(ctx *cli.Context, args []string) error {
			return runMCPProxy(ctx)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Name:         "mcp-profile",
		DefaultValue: "default-mcp",
		Short: i18n.T(
			"MCP profile name for MCP OAuth",
			"MCP OAuth 配置名称 (default-mcp)",
		),
	})

	cmd.Flags().Add(&cli.Flag{
		Name:         "port",
		DefaultValue: "8088",
		Short: i18n.T(
			"Proxy server port",
			"代理服务器端口",
		),
	})

	cmd.Flags().Add(&cli.Flag{
		Name:         "host",
		DefaultValue: "127.0.0.1",
		Short: i18n.T(
			"Proxy server host (use 0.0.0.0 to listen on all interfaces)",
			"代理服务器地址 (使用 0.0.0.0 监听所有网络接口)",
		),
	})

	cmd.Flags().Add(&cli.Flag{
		Name:         "region",
		DefaultValue: "CN",
		Short: i18n.T(
			"Region type: CN or INTL",
			"地域类型: CN 或 INTL",
		),
	})

	cmd.Flags().Add(&cli.Flag{
		Name: "bearer-token",
		Short: i18n.T(
			"Optional static bearer token for client authentication",
			"可选的静态 Bearer Token 用于客户端认证",
		),
	})

	return cmd
}

func runMCPProxy(ctx *cli.Context) error {
	mcpProfileName := ctx.Flags().Get("mcp-profile").GetStringOrDefault("default-mcp")
	portStr := ctx.Flags().Get("port").GetStringOrDefault("8088")
	host := ctx.Flags().Get("host").GetStringOrDefault("127.0.0.1")
	regionStr := ctx.Flags().Get("region").GetStringOrDefault("CN")
	bearerToken := ctx.Flags().Get("bearer-token").GetStringOrDefault("")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %s", portStr)
	}
	var region RegionType
	switch regionStr {
	case "CN":
		region = RegionCN
	case "INTL":
		region = RegionINTL
	default:
		return fmt.Errorf("invalid region: %s, must be CN or INTL", regionStr)
	}

	mcpProfile, err := getOrCreateMCPProfile(ctx, mcpProfileName, region, host, port)
	if err != nil {
		return err
	}

	return startMCPProxy(ctx, mcpProfile, region, host, port, bearerToken)
}

func startMCPProxy(ctx *cli.Context, mcpProfile *McpProfile, region RegionType, host string, port int, bearerToken string) error {
	servers, err := ListMCPServers(ctx, region)
	if err != nil {
		return fmt.Errorf("failed to list MCP servers: %w", err)
	}

	if len(servers) == 0 {
		return fmt.Errorf("no MCP servers found")
	}

	manager := NewOAuthCallbackManager()

	proxy := NewMCPProxy(host, port, region, bearerToken, mcpProfile, servers, manager)
	go proxy.Refresher.Start()

	printProxyInfo(ctx, proxy)

	// 设置信号处理，捕获 Ctrl+C (SIGINT) 和 SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动服务器
	serverErrChan := make(chan error, 1)
	go func() {
		if err := proxy.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// 等待信号或服务器错误
	select {
	case sig := <-sigChan:
		cli.Printf(ctx.Stdout(), "\nReceived signal: %v, shutting down gracefully...\n", sig)
		// 停止 token refresher
		if proxy.Refresher != nil {
			proxy.Refresher.Stop()
		}
		// 停止代理服务器
		if err := proxy.Stop(); err != nil {
			// 如果是超时错误，记录日志但不返回错误，因为服务器已经关闭
			cli.Printf(ctx.Stderr(), "Warning: %v\n", err)
		}
		cli.Println(ctx.Stdout(), "MCP Proxy stopped successfully")
		return nil
	case err := <-serverErrChan:
		return err
	}
}

func printProxyInfo(ctx *cli.Context, proxy *MCPProxy) {
	cli.Printf(ctx.Stdout(), "\nMCP Proxy Server Started\nListen: %s:%d\nRegion: %s\n",
		proxy.Host, proxy.Port, proxy.Region)

	if proxy.BearerToken != "" {
		cli.Println(ctx.Stdout(), "Bearer token: Enabled")
	}

	cli.Println(ctx.Stdout(), "\nAvailable Servers:")
	for _, server := range proxy.McpServers {
		cli.Printf(ctx.Stdout(), "  - %s\n", server.Name)
		if server.Urls.MCP != "" {
			if upstreamURL, err := url.Parse(server.Urls.MCP); err == nil {
				cli.Printf(ctx.Stdout(), "    MCP: http://%s:%d%s\n", proxy.Host, proxy.Port, upstreamURL.Path)
			}
		}
		if server.Urls.SSE != "" {
			if upstreamURL, err := url.Parse(server.Urls.SSE); err == nil {
				cli.Printf(ctx.Stdout(), "    SSE: http://%s:%d%s\n", proxy.Host, proxy.Port, upstreamURL.Path)
			}
		}
	}

	cli.Println(ctx.Stdout(), "\nPress Ctrl+C to stop")
}

func GetContentFromApiResponse(response map[string]any) ([]byte, error) {
	responseBody := response["body"]
	if responseBody == nil {
		return nil, fmt.Errorf("response body is nil")
	}
	switch v := responseBody.(type) {
	case string:
		return []byte(v), nil
	case map[string]any, []any:
		jsonData, _ := json.Marshal(v)
		return jsonData, nil
	case []byte:
		return v, nil
	default:
		return []byte(fmt.Sprintf("%v", v)), nil
	}
}
