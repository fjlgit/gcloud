package utils

import (
	"github.com/skirrund/gcloud/response"

	"strings"

	"github.com/gin-gonic/gin"
)

func CheckQueryParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *gin.Context) bool {
	str := ctx.Query(name)
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckHeaderParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *gin.Context) bool {
	str := ctx.GetHeader(name)
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckParamsWithErrorMsg(name string, str string, v *string, errorMsg string, ctx *gin.Context) bool {
	*v = str
	if len(str) == 0 {
		if len(errorMsg) == 0 {
			ctx.JSON(200, response.ValidateError(name+"不能为空"))
		} else {
			ctx.JSON(200, response.ValidateError(errorMsg))
		}

		return false
	}
	return true
}

func CheckPostFormParamsWithErrorMsg(name string, v *string, errorMsg string, ctx *gin.Context) bool {
	str := ctx.PostForm(name)
	if len(str) == 0 {
		str = ctx.Query(name)
	}
	return CheckParamsWithErrorMsg(name, str, v, errorMsg, ctx)
}

func CheckQueryParams(name string, v *string, ctx *gin.Context) bool {
	return CheckQueryParamsWithErrorMsg(name, v, "", ctx)
}

func CheckPostFormParams(name string, v *string, ctx *gin.Context) bool {
	return CheckPostFormParamsWithErrorMsg(name, v, "", ctx)
}

func CheckHeaderParams(name string, v *string, ctx *gin.Context) bool {
	return CheckHeaderParamsWithErrorMsg(name, v, "", ctx)
}

func SendJSON(ctx *gin.Context, data interface{}) {
	ctx.JSON(200, data)
}

func ShouldBind(ctx *gin.Context, data interface{}) bool {
	err := ctx.ShouldBind(data)
	if err != nil {
		ctx.JSON(200, response.Fail(err.Error()))
		return false
	}
	return true

}

func QueryArray(ctx *gin.Context, name string) []string {
	array := ctx.QueryArray(name)
	var params []string
	if len(array) > 0 {
		for _, a := range array {
			tmp := strings.Split(a, ",")
			params = append(params, tmp...)
		}
	}
	return params
}
func PostFormArray(ctx *gin.Context, name string) []string {
	array := ctx.PostFormArray(name)
	var params []string
	if len(array) > 0 {
		for _, a := range array {
			tmp := strings.Split(a, ",")
			params = append(params, tmp...)
		}
	}
	return params
}
