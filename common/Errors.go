package common

import "errors"

var (
	ERR_LOCK_ALREADY_REQUIRED = errors.New("the lock is in use")

	// 没有物理网卡
	ERR_NO_LOCAL_IP_FOUND = errors.New("can not find network card ip")

	// 断言payload错误
	ERR_ASSERT_CLAIMS_FAIL = errors.New("assert custom claims faild")

	// 获取不到真实token
	ERR_GET_EMPTY_TOKEN = errors.New("can not get real token")

	// 用户名不正确
	ERR_DIFF_USERNAME = errors.New("got different usernames")

	// token已过期
	ERR_TOKEN_EXPIRED = errors.New("token expired")

	// 不可删除当天日志
	ERR_INTRADAY_MUST_SAVE = errors.New("intraday log must preserve")

	// 参数不能为空
	ERR_EMPTY_PARAMS = errors.New("params can not be empty")

	// 参数不合法
	ERR_INVALID_PARAMS = errors.New("params invalid")

	// 数据找不到
	ERR_NOT_FOUND = errors.New("not found any data")

	// 任务执行ip不能为空
	ERR_EMPTY_EXEC_IP = errors.New("job execute ip can not be empty")

	// 类型断言错误
	ERR_ASSERT_WRONG  = errors.New("type assert error")
)
