package main

import (
	"cronTab/config"
	"cronTab/distribute_worker/register"
	"cronTab/distribute_worker/schedule"
	"cronTab/etcd"
	"cronTab/mongodb"
)

//worker 启动
func main() {
	var (
		err error
	)
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
	//初始化注册服务到etcd
	register.InitRegister()

	//初始化任务调度器
	if err = schedule.InitScheduler(); err != nil {
		panic(err)
	}
	select {}
}
