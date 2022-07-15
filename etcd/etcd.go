package etcd

import (
	"fmt"
	"github.com/spf13/viper"
	"go.etcd.io/etcd/client/v3"
	"time"
)

var (
	EtcdKvClient    clientv3.KV
	EtcdLeaseClient clientv3.Lease
	EtcdWatchClient clientv3.Watcher
)

//InitEtcd 初始化etcd连接
func InitEtcd() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
	)
	config = clientv3.Config{
		Endpoints:   viper.GetStringSlice(`etcd.endpoints`),
		DialTimeout: viper.GetDuration(`etcd.dialTimeout`) * time.Millisecond,
	}

	if client, err = clientv3.New(config); err != nil {
		return
	}

	//创建kvClient
	EtcdKvClient = clientv3.NewKV(client)

	//创建leaseClient
	EtcdLeaseClient = clientv3.NewLease(client)

	//创建watchClient
	EtcdWatchClient = clientv3.NewWatcher(client)

	fmt.Println("etcd 初始化成功")
	return nil
}
