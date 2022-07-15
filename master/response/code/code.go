package code

type Code interface {
	GetCodeMsg() string
	GetCode() int
}

//本地localCode 结构体
type localCode struct {
	code    int    // 错误 code
	message string // 此错误code的错误信息
}

//定义一些内置的code

var (
	ParamsError           = localCode{50, "Params Error:参数错误"}
	Ok                    = localCode{20, "ok:请求成功"}
	DataParsingFailed     = localCode{21, "DataParsingFailed:数据解析失败"}
	EtcdPutFailed         = localCode{22, "EtcdPutFailed:etcd 存入数据失败"}
	EtcdDelFailed         = localCode{23, "EtcdDelFailed:etcd 删除数据失败"}
	EtcdGetFailed         = localCode{24, "EtcdGetFailed:etcd 获取数据失败"}
	EtcdCreateLeaseFailed = localCode{25, "EtcdCreateLeaseFailed: 创建租约失败"}
	MongodbSearchFailed   = localCode{26, "MongodbSearchFailed: 数据查询失败"}
)

//New 构造函数
func New(code int, msg string) Code {
	return localCode{
		code:    code,
		message: msg,
	}
}

//GetCodeMsg 获取code 的详细信息
func (c localCode) GetCodeMsg() string {
	return c.message
}

//GetCode 获取code码
func (c localCode) GetCode() int {
	return c.code
}
