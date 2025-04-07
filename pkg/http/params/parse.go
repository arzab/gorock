package params

import (
	"errors"
	"fmt"
	"github.com/arzab/gorock/pkg/http/responses"
	"github.com/gofiber/fiber/v2"
	"mime/multipart"
	"net/http"
	"reflect"
)

func DefaultHandler[T any, pointer Service[T]](key ...string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var paramsObj pointer = new(T)

		err := ParseRequest(ctx, paramsObj)
		if err != nil {
			return fmt.Errorf("failed ot parse request params: %v", err)
		}

		err = paramsObj.Validate(ctx)
		if err != nil {
			var errorResponse *responses.ErrorResponse

			if errors.As(err, &errorResponse) {
				return errorResponse
			} else {
				return responses.NewError(http.StatusBadRequest, err.Error())
			}
		}

		if len(key) > 0 {
			ctx.Locals(key[0], paramsObj)
		} else {
			ctx.Locals("params", paramsObj)
		}

		return ctx.Next()
	}
}

// ParseRequest Parsing all request params by one function to the one struct by tags, struct must be pointer
//
// Parsing headers by `reqHeader:"param"`
//
// Parsing multipart form data by `form:"param"`
//
// Parsing query params by  `query:"param"`
//
// Parsing body by tags json/yaml etd json:"param"
//
// Parsing multipart use data type *multipart.FileHeader and tag `form:"param"`
func ParseRequest(ctx *fiber.Ctx, params interface{}) error {
	if reflect.TypeOf(params).Kind() != reflect.Ptr {
		return fmt.Errorf("params must be pointer")
	}

	//Парсинг query параметров через теги `query:"param"`
	err := ctx.QueryParser(params)
	if err != nil {
		return fmt.Errorf("failed parse query params: %v", err)
	}

	if ctx.Method() != "GET" && len(ctx.Body()) > 0 {
		err = ctx.BodyParser(params)
		if err != nil {
			return fmt.Errorf("failed parse body: %v", err)
		}

		// парсим файлы из multipart
		err = parseMultipartFiles(ctx, params)
		if err != nil {
			return fmt.Errorf("failed parse multipart files: %v", err)
		}
	}

	//Парсинг header параметров через теги `reqHeader:"param"`
	err = ctx.ReqHeaderParser(params)
	if err != nil {
		return fmt.Errorf("failed parse headers: %v", err)
	}

	//Парсинг url (/path/:param1/:param2) параметров через теги `param:"param"`
	err = ctx.ParamsParser(params)
	if err != nil {
		return fmt.Errorf("failed parse url params: %v", err)
	}

	return nil
}

func parseMultipartFiles(ctx *fiber.Ctx, out interface{}) error {
	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("out must be a non-nil pointer")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("out must point to a struct")
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldType := typ.Field(i)
		tagValue := fieldType.Tag.Get("form")
		if tagValue == "" {
			continue
		}

		file, err := ctx.FormFile(tagValue)
		if err != nil {
			continue
		}

		field := val.Field(i)
		if field.CanSet() && field.Kind() == reflect.Ptr && field.Type().Elem() == reflect.TypeOf(multipart.FileHeader{}) {
			field.Set(reflect.ValueOf(file))
		}
	}
	return nil
}
