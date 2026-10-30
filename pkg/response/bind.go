package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var trans ut.Translator

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name == "" {
				name = strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
			}
			if name == "" {
				name = fld.Name
			}
			return name
		})

		zhLocale := zh.New()
		uni := ut.New(zhLocale, zhLocale)
		trans, _ = uni.GetTranslator("zh")
		_ = zh_translations.RegisterDefaultTranslations(v, trans)
	}
}

// FormatBindError 将参数绑定或校验错误转换为友好的中文提示
func FormatBindError(err error) string {
	if err == nil {
		return ""
	}

	// 1. 请求体为空
	if errors.Is(err, io.EOF) {
		return "请求体不能为空"
	}

	// 2. 字段类型不匹配 (如期望数值类型却传入字符串)
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		if typeErr.Field != "" {
			return fmt.Sprintf("参数 [%s] 数据类型错误，期望为 %s 类型", typeErr.Field, typeErr.Type)
		}
		return fmt.Sprintf("参数数据类型错误，期望为 %s 类型", typeErr.Type)
	}

	// 3. JSON 语法解析错误
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Sprintf("JSON 语法错误 (偏移量: %d)", syntaxErr.Offset)
	}

	// 4. Gin Validator 结构体标签校验错误
	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		if trans != nil {
			var errMsgs []string
			for _, fe := range valErrs {
				msg := fe.Translate(trans)
				errMsgs = append(errMsgs, msg)
			}
			if len(errMsgs) > 0 {
				return "参数校验失败: " + strings.Join(errMsgs, "; ")
			}
		}
	}

	// 5. 原生字符串转译兜底
	msg := err.Error()
	if strings.Contains(msg, "EOF") {
		return "请求体不能为空或内容不完整"
	}
	if strings.Contains(msg, "cannot unmarshal") {
		return "参数数据类型不匹配"
	}
	if strings.Contains(msg, "validation for") {
		return "参数校验失败，请检查必填项与格式规范"
	}

	return msg
}

// ParamError 快捷处理参数绑定与校验错误并以 400 响应
func ParamError(c *gin.Context, err error) {
	BadRequest(c, FormatBindError(err))
}
