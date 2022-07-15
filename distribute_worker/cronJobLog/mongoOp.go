package cronJobLog

import (
	"cronTab/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"time"
)

//MongoLog 日志通道
type MongoLog struct {
	collection *mongo.Collection
	LogChan    chan *JobLog
	logBuf     []interface{}
	count      int
}

var MLog *MongoLog

func init() {
	MLog = &MongoLog{
		LogChan: make(chan *JobLog),
		logBuf:  make([]interface{}, 0, 20), //容量20
		count:   0,
	}
	go MLog.insertToMongodb()
}

func (ml *MongoLog) PushLog(log *JobLog) {
	ml.LogChan <- log
}

func (ml *MongoLog) insertToMongodb() {
	//超时自动提交
	commitTimer := time.NewTimer(2 * time.Second)

	for {
		select {
		case log := <-ml.LogChan:
			ml.logBuf = append(ml.logBuf, log)
			ml.count++
			if ml.count == 20 {
				mongodb.MongoClient.Database("cron").Collection("log").
					InsertMany(context.Background(), ml.logBuf)
				//清空缓存
				ml.logBuf = []interface{}{}
				ml.count = 0
			}
		case <-commitTimer.C:
			if ml.count != 0 {
				mongodb.MongoClient.Database("cron").Collection("log").
					InsertMany(context.Background(), ml.logBuf)
				//清空缓存
				ml.logBuf = []interface{}{}
				ml.count = 0
			}
			//重置时间
			commitTimer.Reset(2 * time.Second)
		}
	}
}
