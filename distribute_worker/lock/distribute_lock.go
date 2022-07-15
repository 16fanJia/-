package lock

import (
	"cronTab/etcd"
	"errors"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
)

//分布式锁

const (
	DisTributeLock = "/lock/"
)

var (
	DLockAlreadyExists = errors.New("锁已经被占用！")
)

type DLock struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	jobName    string
	cancelFunc context.CancelFunc
	leaseId    clientv3.LeaseID
	locked     bool
}

//NewDLock 分布式锁构造函数
func NewDLock(jobName string) *DLock {
	return &DLock{
		kv:      etcd.EtcdKvClient,
		lease:   etcd.EtcdLeaseClient,
		jobName: jobName,
	}
}

//TryDLock 尝试获取分布式锁
func (dl *DLock) TryDLock() error {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		err            error
		keepAliveChan  <-chan *clientv3.LeaseKeepAliveResponse
		txnResponse    *clientv3.TxnResponse
	)
	//创建租约
	if leaseGrantResp, err = dl.lease.Grant(context.TODO(), 5); err != nil {
		return err
	}
	//租约id
	dl.leaseId = leaseGrantResp.ID

	ctx, cancelFunc := context.WithCancel(context.Background())
	dl.cancelFunc = cancelFunc
	//续约租约
	if keepAliveChan, err = dl.lease.KeepAlive(ctx, dl.leaseId); err != nil {
		cancelFunc()                                //取消续租
		dl.lease.Revoke(context.TODO(), dl.leaseId) //释放租约
		return err
	}

	//处理租约应答的协程
	go func() {
		for {
			select {
			case keeResp := <-keepAliveChan:
				if keeResp == nil {
					fmt.Println("租约失效")
					goto END
				} else {
					fmt.Println("收到自动续租应答", keeResp.ID)
				}
			}
		}
	END:
	}()

	//创建事物
	txn := dl.kv.Txn(context.Background())

	lockKey := DisTributeLock + dl.jobName

	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "yes", clientv3.WithLease(dl.leaseId))).
		Else(clientv3.OpGet(lockKey))

	if txnResponse, err = txn.Commit(); err != nil {
		cancelFunc()                                //取消续租
		dl.lease.Revoke(context.TODO(), dl.leaseId) //释放租约
		return err
	}
	//抢锁不成功
	if !txnResponse.Succeeded {
		return DLockAlreadyExists
	}
	dl.locked = true
	return nil
}

//UnLock 解锁
func (dl *DLock) UnLock() {
	if dl.locked { //只有上锁成功 才能释放锁
		dl.cancelFunc()
		dl.lease.Revoke(context.Background(), dl.leaseId)
	}
}
