/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package rpc
 *@file    client
 *@date    2026/3/25 14:46
 */

package brpc

import (
	"StarRocksProbe/proto"
	"StarRocksProbe/util"
	"context"
	"fmt"
	"github.com/patrickmn/go-cache"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func attachToken(ctx context.Context, token string) context.Context {
	md := metadata.New(map[string]string{"authorization": "Bearer " + token, "ip": fmt.Sprintf("%s:%d", util.H.Ip, util.Read.RPC.Port)})
	return metadata.NewOutgoingContext(ctx, md)
}

func AgentSchemaItem(input string) []string {
	conn, err := grpc.Dial(rpcService, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		util.Loggrs.Warnf("%v%v", rpclabel, err.Error())
		return nil
	}
	defer conn.Close()

	client := proto.NewDataServiceClient(conn)

	util.Loggrs.Infof("%v exec, rpc请求数据，request:[%d]", rpclabel, len(input))

	// 关键修改：创建带 token 的上下文
	var ctx context.Context
	if val, ok := lastcache.Get("ctx"); ok {
		ctx = val.(context.Context)
	} else {
		token := "FBLmMptfqRMgJmkp0dgJ6A==" // 需要替换为实际的 token
		ctx = attachToken(context.Background(), token)
	}
	// 载入30分钟的缓存
	lastcache.Set("ctx", ctx, cache.DefaultExpiration)

	// 使用带 token 的上下文调用 RPC 方法
	response, err := client.SchemaItem(ctx, &proto.SchemaRegexRequest{
		Stmt: input,
		Ip:   fmt.Sprintf("%s:%d", util.H.Ip, util.Read.RPC.Port),
	})
	if err != nil {
		util.Loggrs.Warnf("%v %v", rpclabel, err.Error())
		return nil
	}
	util.Loggrs.Infof("%v succ, rpc请求数据，request:[%d] response:[%d]", rpclabel, len(input), len(response.Stmt))
	return response.Stmt
}
