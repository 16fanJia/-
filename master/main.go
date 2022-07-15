package main

import (
	"cronTab/config"
	"cronTab/etcd"
	"cronTab/master/router"
	"cronTab/mongodb"
	"github.com/gin-gonic/gin"
	"runtime"
)

func main() {
	var (
		ginRouter *gin.Engine
		err       error
	)

	ginRouter = gin.Default()
	//配置线程
	runtime.GOMAXPROCS(runtime.NumCPU())
	//初始化配置文件
	if err = config.InitConfig(); err != nil {
		panic(err)
	}
	//初始化mongodb
	if err = mongodb.InitMongoDB(); err != nil {
		panic(err)
	}
	//初始化etcd
	if err = etcd.InitEtcd(); err != nil {
		panic(err)
	}
	//注册Master相关API
	router.RegisterMasterApi(ginRouter)

	if err = ginRouter.Run("localhost:8070"); err != nil {
		panic(err)
		return
	}
}
