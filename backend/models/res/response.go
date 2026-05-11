package res

import (
	"context"
	"net/http"
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	Code int    `json:"code"`
	Data any    `json:"data"`
	Msg  string `json:"msg"`
}
type ListResponse[T any] struct {
	Count int64 `json:"count"`
	List  any   `json:"list"`
}

const (
	Success = 0
	Error   = 7
)

func Result(code int, data any, msg string, c context.Context, ctx *app.RequestContext) {
	resp := Response{
		Code: code,
		Data: data,
		Msg:  msg,
	}
	// 使用sonic序列化JSON
	jsonData, err := sonic.Marshal(resp)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "序列化响应失败")
		return
	}
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", jsonData)
}

func Ok(data any, msg string, c context.Context, ctx *app.RequestContext) {
	Result(Success, data, msg, c, ctx)
}
func OkWithData(data any, c context.Context, ctx *app.RequestContext) {
	Result(Success, data, "成功", c, ctx)
}
func OkWithMessage(msg string, c context.Context, ctx *app.RequestContext) {
	Result(Success, map[string]any{}, msg, c, ctx)
}
func OkWith(c context.Context, ctx *app.RequestContext) {
	Result(Success, map[string]any{}, "成功", c, ctx)
}

func OkWithList(list any, count int64, c context.Context, ctx *app.RequestContext) {
	OkWithData(ListResponse[any]{
		List:  list,
		Count: count,
	}, c, ctx)
}

func Fail(data any, msg string, c context.Context, ctx *app.RequestContext) {
	Result(Error, data, msg, c, ctx)
}
func FailWithMessage(msg string, c context.Context, ctx *app.RequestContext) {
	Result(Error, map[string]any{}, msg, c, ctx)
}

func FailWithError(err error, obj any, c context.Context, ctx *app.RequestContext) {
	msg := GetValidMsg(err, obj)
	FailWithMessage(msg, c, ctx)
}

func FailWithCode(code ErrorCode, c context.Context, ctx *app.RequestContext) {
	msg, ok := ErrorMap[code]
	if ok {
		Result(int(code), map[string]any{}, msg, c, ctx)
		return
	}
	Result(Error, map[string]any{}, "未知错误", c, ctx)
}

// GetValidMsg 返回结构体中的msg参数
func GetValidMsg(err error, obj any) string {
	// 使用的时候，需要传obj的指针
	getObj := reflect.TypeOf(obj)
	// 将err接口断言为具体类型
	if errs, ok := err.(validator.ValidationErrors); ok {
		// 断言成功
		for _, e := range errs {
			// 循环每一个错误信息
			// 根据报错字段名，获取结构体的具体字段
			if f, exits := getObj.Elem().FieldByName(e.Field()); exits {
				msg := f.Tag.Get("msg")
				return msg
			}
		}
	}

	return err.Error()
}
