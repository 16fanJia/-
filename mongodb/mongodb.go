package mongodb

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

//MongoClient 全局的mongodb 客户端
var MongoClient *mongo.Client

//InitMongoDB 初始化mongoDB
func InitMongoDB() (err error) {
	//连接参数
	clientUrlOp := options.Client().ApplyURI(viper.GetString(`mongodb.url`))

	clientTimeOutOp := options.Client().SetConnectTimeout(5 * time.Second)

	if MongoClient, err = mongo.Connect(context.TODO(), clientUrlOp, clientTimeOutOp); err != nil {
		return
	}

	if err = MongoClient.Ping(context.TODO(), nil); err != nil {
		return
	}

	fmt.Println("mongodb 数据库初始化成功")
	return nil
}
