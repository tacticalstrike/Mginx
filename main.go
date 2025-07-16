package main

import (
	"Mginx/LoadBanlance"
	"Mginx/config"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

func main() {
	servers, err := config.LoadConfd()
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}           // 等待每个server程序退出（虽然不应该退出）
	for _, server := range servers { // 对于每个从配置文件解析出的server
		if server.Listen == 0 {
			panic(fmt.Sprintf("Invalid server listen address in server %s", server.Name))
		}

		srv := EstablishServer(server)                   // 我们从配置文件中的条目来搭建server，主要是建立多路复用器
		if server.SSLCert == "" && server.SSLKey == "" { // 没有配置SSL就普通监听
			go func() {
				defer wg.Done()
				err := srv.ListenAndServe()
				if err != nil {
					panic(err)
				}
			}()
			wg.Add(1)
		} else if server.SSLCert != "" && server.SSLKey != "" { // 配置了SSL就监听TLS
			go func() {
				defer wg.Done()
				err := srv.ListenAndServeTLS(server.SSLCert, server.SSLKey)
				if err != nil {
					panic(err)
				}
			}()
			wg.Add(1)
		} else {
			panic(fmt.Sprintf("Invalid server SSL cert or server key in server %s", server.Name))
		}
	}
	wg.Wait()
}

func EstablishServer(server *config.Server) *http.Server {
	logFile, err := os.OpenFile(server.Name, os.O_RDWR|os.O_CREATE, 0644)
	defer func(logFile *os.File) {
		err = logFile.Close()
		if err != nil {
			panic(err)
		}
	}(logFile)
	if err != nil {
		panic(err)
	}
	errorLog := log.New(logFile, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	mux := http.NewServeMux()
	for _, location := range server.Location {
		if location.Static != "" && location.Forward != nil { //forward字段和static字段冲突
			panic(fmt.Sprintf("Forward conflicts with Static in server %s", server.Name))
		}

		if location.Static != "" { //如果static字段不为空，就建立文件服务器
			staticFile := http.StripPrefix(location.Path, http.FileServer(http.Dir(location.Static)))
			mux.Handle(location.Path, staticFile)
		} else if location.Forward != nil { //如果forward字段不为空，就解析该字段并建立负载均衡器
			upstream, err := ParseForward(location.Forward)
			if err != nil {
				panic(err)
			}
			mux.Handle(location.Path, LoadBanlance.ForwardTo(upstream))
			go func() {
				errChan := make(chan error, 10)
				LoadBanlance.KeepAlive(upstream, errChan)
				for err := range errChan {
					errorLog.Println(err)
				}
			}()
		} else { //既没有forward字段也没有static字段
			panic(fmt.Sprintf("Location of server %s must have forward or static", server.Name))
		}
	}

	srv, err := NewServer(
		"0.0.0.0",
		WithPort(strconv.Itoa(server.Listen)),
		WithHandler(mux),
		WithErrorLog(errorLog),
	)
	if err != nil {
		panic(err)
	}

	return srv
}

func ParseForward(forward []string) (*LoadBanlance.UpStream, error) {
	upstream := LoadBanlance.UpStream{
		Algorithm: "",
		Address:   make([]string, 0),
		Weights:   make([]int, 0),
	}

	upstream.Algorithm = forward[0]
	if upstream.Algorithm == "weight" {
		for _, str := range forward[1:] {
			str = strings.TrimSpace(str)            //先去除字符串两边空格
			add, wt, found := strings.Cut(str, " ") //然后再提取由空格分开的IP和weight
			upstream.Address = append(upstream.Address, add)
			if found {
				wt, err := strconv.Atoi(strings.TrimPrefix(wt, "weight="))
				if err != nil {
					return nil, fmt.Errorf("parse forward table failed: %s", err.Error())
				}
				upstream.Weights = append(upstream.Weights, wt)
			} else {
				upstream.Weights = append(upstream.Weights, 1)
			}
		}
	} else {
		for _, str := range forward[1:] {
			upstream.Address = append(upstream.Address, str)
		}
	}
	return &upstream, nil
}
