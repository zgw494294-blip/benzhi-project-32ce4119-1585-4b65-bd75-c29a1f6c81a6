package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type config struct {
	addr      string
	database  string
	selfcheck bool
}

func parseConfig(args []string) (config, error) {
	defaultAddr := "127.0.0.1:19081"
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1024 || number > 65535 {
			return config{}, fmt.Errorf("PORT 必须是 1024-65535 的端口号")
		}
		defaultAddr = net.JoinHostPort("127.0.0.1", port)
	}
	set := flag.NewFlagSet("server", flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	addr := set.String("addr", defaultAddr, "回环监听地址")
	database := set.String("db", "data/wetland-release.db", "SQLite 数据库路径")
	selfcheck := set.Bool("selfcheck", false, "运行真实 HTTP 业务自检后退出")
	if err := set.Parse(args); err != nil {
		return config{}, err
	}
	if set.NArg() != 0 {
		return config{}, fmt.Errorf("存在未识别参数")
	}
	normalized, err := validateAddr(*addr)
	if err != nil {
		return config{}, err
	}
	return config{addr: normalized, database: *database, selfcheck: *selfcheck}, nil
}

func validateAddr(value string) (string, error) {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return "", fmt.Errorf("-addr 必须为 host:port：%w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("仅允许监听回环 IP 地址")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1024 || number > 65535 {
		return "", fmt.Errorf("监听端口必须在 1024-65535 之间")
	}
	return net.JoinHostPort(ip.String(), port), nil
}
