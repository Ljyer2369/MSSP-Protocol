// something about broadcast

package networks

import (
	"log"
	"net"
	"sync"
)

var connMaplock sync.Mutex                        //互斥锁
var connectionPool = make(map[string]net.Conn, 0) //

func TcpDial(context []byte, addr string) { //TcpDial函数用于建立TCP连接
	var conn net.Conn
	connMaplock.Lock() //加锁
	defer connMaplock.Unlock()
	if connectionPool[addr] == nil { //如果连接池中没有该连接，则建立连接
		conn, err := net.Dial("tcp", addr) //建立TCP连接
		if err != nil {
			log.Println("connect error", err)
			return
		}
		connectionPool[addr] = conn //将连接存入连接池
	}
	conn = connectionPool[addr] //如果连接池中有该连接，则直接使用

	_, err := conn.Write(append(context, '\n')) //发送消息
	if err != nil {
		return
	}
}

func Broadcast(sender string, receivers []string, msg []byte) { //Broadcast函数用于广播消息
	for _, ip := range receivers { //遍历接收者
		if ip == sender { //如果遍历到的接收者为发送者，则跳过
			continue
		}
		go TcpDial(msg, ip) //建立TCP连接
	}
}

func CloseAllConnInPool() { //CloseAllConnInPool函数用于关闭连接池中的所有连接
	for _, conn := range connectionPool {
		conn.Close()
	}
}

// todo
// long connect (not close immediately) ...
