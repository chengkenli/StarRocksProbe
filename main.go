/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package StarRocksProbe
 *@file    main
 *@date    2025/2/12 14:02
 */

package main

import (
	"StarRocksProbe/app"
	_ "StarRocksProbe/init"
	"StarRocksProbe/util"
)

func main() {
	util.Init()
	app.App()
}
