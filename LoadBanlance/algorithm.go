package LoadBanlance

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"math"
	"slices"
	"strconv"
)

func RoundRobin(upStream *UpStream, counter int) (ip string) {
	length := len(upStream.Address)
	index := counter % length
	return upStream.Address[index]
}

func Weight(upStream *UpStream, counter int) (ip string) {
	for index, weight := range upStream.Weights {
		if counter < weight {
			return upStream.Address[index]
		}
	}
	return ""
}

type dA struct {
	addr string
	hash int
}

// StepSize是计算哈希环时，虚拟节点的间距，默认为2^32/16=268435456，即最多有16个虚拟节点均匀分布在哈希环上
const StepSize int = 268435456

func Hash(upStream *UpStream, clientAddr string) (ip string, err error) {
	i, err := strconv.ParseInt(fmt.Sprintf("%x", md5.Sum([]byte(clientAddr)))[:8], 16, 64)
	if err != nil {
		return "", err
	}
	var srcAddr int = int(i) % int(math.Pow(2, 32)-1)
	var destAddr []dA
	for _, addr := range upStream.Address {
		// 取生成哈希值的前8位转为数字，所以取值范围为0~2^32-1
		i, err := strconv.ParseInt(fmt.Sprintf("%x", md5.Sum([]byte(addr)))[:8], 16, 64)
		if err != nil {
			return "", err
		}
		destAddr = append(destAddr, dA{addr: addr, hash: int(i)})
	}
	slices.SortFunc(destAddr, func(a, b dA) int { return cmp.Compare(a.hash, b.hash) })
	numOfAddr := len(destAddr)
	for range 3 { // 这里的数字是构建哈希环的迭代次数，值不应该太大
		for i := 0; i < numOfAddr; i++ { // 每次都在哈希环中添加等同于目的IP数量的节点
			destAddr = append(destAddr,
				dA{
					addr: destAddr[i].addr,
					hash: (destAddr[numOfAddr+i-1].hash + StepSize) % int(math.Pow(2, 32)-1),
				})
		}
	}
	slices.SortFunc(destAddr, func(a, b dA) int { return cmp.Compare(a.hash, b.hash) })
	for _, addr := range destAddr {
		if addr.hash >= srcAddr {
			return addr.addr, nil
		}
	}
	return destAddr[0].addr, nil
}
