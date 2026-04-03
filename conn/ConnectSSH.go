/*
 *@author  chengkenli
 *@project pjstack
 *@package main
 *@file    ConnectSSH
 *@date    2024/9/2 13:58
 */

package conn

import (
	"StarRocksProbe/util"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ConnectSSH SSH函数
func ConnectSSH(c *util.ConnSSH) interface{} {
	// 连接到SSH服务器
	client, err := connectSSH(c.User, c.Host, c.Port, PublicKeyAuthFunc(c.PrivateKeyFile))
	if err != nil {
		fmt.Println(err.Error())
	}
	defer func(client *ssh.Client) {
		err := client.Close()
		if err != nil {
			fmt.Println("=====>", err.Error())
			return
		}
	}(client)
	// 执行命令
	output := runCommand(client, c.Command)
	return output
}

// PublicKeyAuthFunc 连接到SSH服务器
func PublicKeyAuthFunc(kPath string) ssh.AuthMethod {
	// 如果路径以 ~ 开头，展开为家目录
	if kPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("获取家目录失败:", err)
			return nil
		}
		kPath = filepath.Join(home, kPath[2:])
	}
	// 读取私钥文件
	key, err := ioutil.ReadFile(kPath)
	if err != nil {
		fmt.Println(err.Error())
	}
	// 创建SSH签名器
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		fmt.Println(err.Error())
	}

	return ssh.PublicKeys(signer)
}

// connectSSH 建立SSH客户端连接
func connectSSH(user, host string, port int, auth ssh.AuthMethod) (*ssh.Client, error) {
	var addr = fmt.Sprintf("%s:%d", host, port)
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 在生产环境中应该更安全地处理HostKey
		//HostKeyCallback: ssh.FixedHostKey(signer.PublicKey()),
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// runCommand 执行远程命令
func runCommand(client *ssh.Client, command string) string {
	sess, err := client.NewSession()
	if err != nil {
		fmt.Println(1, err.Error())
	}
	defer sess.Close()

	output, _ := sess.CombinedOutput(command)
	//if err != nil {
	//	fmt.Println(2,err.Error())
	//}
	return string(output)
}
