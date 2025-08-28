package process

import (
	"fmt"
	"net"  // 引入 Go 的网络标准库
	"time" // 引入时间库，用于设置超时
)

// CheckPort 函数尝试连接一个指定的 TCP 端口，以确定是否有进程在监听它。
// port: 需要检查的端口号。
// 返回值: 如果端口在线则返回 true，否则返回 false。
func CheckPort(port int) bool {
	// 构建目标地址字符串，格式为 "host:port"
	// 我们检查本地主机 127.0.0.1
	address := fmt.Sprintf("127.0.0.1:%d", port)
	// 使用 net.DialTimeout 尝试建立一个 TCP 连接。
	// "tcp": 网络类型。
	// address: 目标地址。
	// 1 * time.Second: 设置一个1秒的超时。如果1秒内无法建立连接，函数就会返回一个超时错误。
	// 这是一个非常重要的设置，可以防止我们的程序在检查一个无响应的端口时卡住太久。
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)

	// 检查是否发生错误
	if err != nil {
		// 如果有错误（例如“连接被拒绝”或“超时”），
		// 就意味着端口是离线的。
		return false
	}

	// 如果 err 为 nil，意味着连接成功建立。
	// 这说明端口是在线的。
	// 我们需要立即关闭这个刚刚建立的连接，因为它只是用于探测。
	defer conn.Close() // defer 关键字确保 conn.Close() 会在函数返回前被执行。

	return true
}
