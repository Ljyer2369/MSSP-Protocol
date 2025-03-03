package utils

import (
	"blockEmulator/params"
	"log"
	"strconv"
)

// the default method
func Addr2Shard(addr Address) int { //Addr2Shard方法用于将地址转换为分片ID
	last16_addr := addr[len(addr)-8:]
	num, err := strconv.ParseUint(last16_addr, 16, 64)
	if err != nil {
		log.Panic(err)
	}
	return int(num) % params.ShardNum //返回地址对应的分片ID
}
