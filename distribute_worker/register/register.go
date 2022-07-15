package register

import (
	"cronTab/common"
	"cronTab/etcd"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"net"
)

var (
	NotFindLocalIp = errors.New("未找到本机ip地址！")
)

//服务注册  worker将自己的ip 注册到etcd 的/cron/workers/ 目录下

type Registration struct {
	kv      clientv3.KV
	lease   clientv3.Lease //租约客户端
	localIp string         //本机ip地址
}

var Register *Registration

func InitRegister() {
	Register = &Registration{
		kv:    etcd.EtcdKvClient,
		lease: etcd.EtcdLeaseClient,
	}

	ip, _ := Register.getLocalIp()
	Register.localIp = ip

	go Register.registerAndKeepOnline()
}

//registerAndKeepOnline 服务注册到etcd /cron/workers/目录下 并且保持续租
func (r *Registration) registerAndKeepOnline() {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		err            error
		leaseId        clientv3.LeaseID
		leaseChan      <-chan *clientv3.LeaseKeepAliveResponse
	)
RETRY: //重试机制
	//创建租约
	if leaseGrantResp, err = r.lease.Grant(context.Background(), 5); err != nil {
		goto RETRY
	}
	//租约ID
	leaseId = leaseGrantResp.ID

	//续租 节点宕机 会自动删除key
	if leaseChan, err = r.lease.KeepAlive(context.TODO(), leaseId); err != nil {
		goto RETRY
	}

	key := common.EtcdWorkerKey + r.localIp
	ctx, cancelFunc := context.WithCancel(context.Background())

	//租约关联
	if _, err = r.kv.Put(ctx, key, r.localIp, clientv3.WithLease(leaseId)); err != nil {
		cancelFunc() //取消续租
		r.lease.Revoke(context.Background(), leaseId)
		goto RETRY
	}

	//循环一直处理 续租应答
	for {
		select {
		case keepAliveResp := <-leaseChan:
			if keepAliveResp == nil {
				fmt.Println("租约失效")
				goto RETRY
			}
		}
	}
}

//获取本地ip
func (r *Registration) getLocalIp() (string, error) {
	var (
		addrSlice []net.Addr
		err       error
	)
	if addrSlice, err = net.InterfaceAddrs(); err != nil {
		return "", err
	}

	for _, address := range addrSlice {
		// 检查ip地址判断是否回环地址
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.To4().String(), nil
			}
		}
	}
	//如果未返回 则未找到网卡
	err = NotFindLocalIp
	return "", err
}
