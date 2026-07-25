/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package brpc
 *@file    init
 *@date    2026/3/25 15:56
 */

package brpc

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/patrickmn/go-cache"
	"time"
)

var rpcService, rpclabel string
var lastcache = cache.New(30*time.Minute, 30*time.Minute)

func init() {
	var host string
	var port int
	if util.Read.RPC.Server == "" {
		host = util.H.Ip
	} else {
		host = util.Read.RPC.Server
	}
	if host == "" {
		host = "127.0.0.1"
	}
	rpclabel = fmt.Sprintf("[grpc]:")
	if util.Read.RPC.Port <= 0 {
		util.Loggrs.Errorf("%v 端口丢失，rpc服务连接异常！", rpclabel)
	} else {
		port = util.Read.RPC.Port
	}
	rpcService = fmt.Sprintf("%s:%d", host, port)
	if rpcService == "" {
		util.Loggrs.Fatalf("%v rpc server failed.", rpclabel)
	}
	rpclabel = fmt.Sprintf("[grpc] %v:", rpcService)
}
