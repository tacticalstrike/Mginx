package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Location struct {
	Path    string   `json:"path"`
	Static  string   `json:"static"`
	Forward []string `json:"forward"`
}
type Server struct {
	Listen   int        `json:"listen"`
	Name     string     `json:"name"`
	SSLCert  string     `json:"ssl_cert"`
	SSLKey   string     `json:"ssl_key"`
	Location []Location `json:"location"`
}

func LoadConfig(filePath string) (*Server, error) {
	var server Server
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &server)
	if err != nil {
		return nil, err
	}
	return &server, nil
}

func LoadConfd() ([]*Server, error) {
	var servers []*Server
	root := "./conf.d" // 指定要遍历的目录

	// 遍历目录并处理JSON文件
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和非JSON文件
		if info.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		// 处理找到的JSON文件
		fmt.Println("Found ", path)
		server, err := LoadConfig(path) //加载每个找到的json格式配置文件
		if err != nil {
			return err
		}
		servers = append(servers, server)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return servers, nil
}
