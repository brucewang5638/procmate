package main

import "hk-console/cmd"

func main() {
	// main 函数的唯一职责就是执行 cmd 包中的根命令
	cmd.Execute()
}
