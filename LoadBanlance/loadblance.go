package LoadBanlance

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type UpStream struct {
	Algorithm string //负载均衡算法，取值为“round-robin”(轮询)、"weight"(加权)、"hash"（哈希）
	Address   []string
	Weights   []int
}

func ForwardTo(stream *UpStream) http.Handler {
	if stream == nil {
		return nil
	}
	var Counter int = 0
	mutex := sync.Mutex{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, err := CheckCache(r.URL.String()); ok {
			err = SendCache(w, r.URL.String())
			if err != nil {
				fmt.Println(err)
			}
			return
		} else {
			fmt.Println(err)
		}
		var ip string
		mutex.Lock() //由于golang的每个请求都会单独开一个goroutine，这里会产生竞争问题
		switch stream.Algorithm {
		case "round-robin":
			ip = RoundRobin(stream, Counter)
			Counter++
			if Counter >= len(stream.Address) {
				Counter = 0
			}
		case "weight":
			ip = Weight(stream, Counter)
			Counter++
			if Counter >= len(stream.Weights) {
				Counter = 0
			}
		case "hash":
			var err error
			ip, err = Hash(stream, r.RemoteAddr)
			if err != nil {
				fmt.Println(err)
				return
			}
		default:
			panic("Unknown algorithm")
		}
		mutex.Unlock()

		fmt.Println(r.Method, r.URL.Path, "Foward to", ip)
		if ip == "" {
			http.Error(w, "Config error at server", http.StatusInternalServerError)
			panic("No IP")
		}

		resp, err := ForwardRequestTo(ip, r) // 把http请求“r”，转发到“ip”去
		if err != nil {
			http.Error(w, "IP resolve error at server", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		err = StoreCache(resp.Body, r.URL.String())
		if err != nil {
			fmt.Println(err) // 缓存问题不需要阻止程序
		}

		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
}

func ForwardRequestTo(ip string, r *http.Request) (*http.Response, error) {
	newReq, err := http.NewRequest(r.Method, ip+r.URL.Path, r.Body)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			//这个配置只针对于测试用的SSL证书
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(newReq)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
