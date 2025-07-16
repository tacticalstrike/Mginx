package LoadBanlance

import (
	"net"
	"strings"
	"sync"
	"time"
)

func KeepAlive(stream *UpStream, errChan chan<- error) {
	defer close(errChan)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, address := range stream.Address {
		go func() {
			ticker := time.NewTicker(5 * time.Second) // 每5秒对该地址进行一次心跳检测
			defer ticker.Stop()
			for {
				//fmt.Println("Detecting", address)
				ok := stream.ContainsAddr(address) // 检查该地址是否还在列表中
				select {
				case <-ticker.C:
					conn, err := net.Dial("tcp", stream.CutHTTPPrefix(address))
					mu.Lock()
					if err != nil {
						errChan <- err
						if ok { // 如果主机不存活，并且该地址未被移除（主机突然异常）
							stream.RemoveAddr(address) // 那么移除它
						}
						// 如果主机不存活，并且该地址已被移除（主机一直异常），那么conn就是nil
					} else {
						if !ok { // 如果主机存活，并且该地址已被移除（主机恢复正常）
							// 那么添加它
							stream.AddAddr(address)
						}
						// 如果主机存活，并且该地址没有被移除（主机一直正常）
						conn.Close()
					}
					mu.Unlock()
				}
			}

		}()
		wg.Add(1)
	}
	wg.Wait()
}

// httpAddr是配置文件中格式为http://ip:port或https://ip:port的字符串
func (stream *UpStream) CutHTTPPrefix(addr string) string { //
	if strings.HasPrefix(addr, "http://") {
		addr, _ = strings.CutPrefix(addr, "http://")
	} else if strings.HasPrefix(addr, "https://") {
		addr, _ = strings.CutPrefix(addr, "https://")
	}

	return addr
}

func (stream *UpStream) ContainsAddr(addr string) bool {
	for _, address := range stream.Address {
		if address == addr {
			return true
		}
	}
	return false
}

func (stream *UpStream) RemoveAddr(addr string) {
	for index, address := range stream.Address {
		if address == addr {
			stream.Address = append(stream.Address[:index], stream.Address[index+1:]...)
		}
	}
}

func (stream *UpStream) AddAddr(addr string) {
	stream.Address = append(stream.Address, addr)
}
