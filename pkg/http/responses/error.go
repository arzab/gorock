package responses

import (
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Source  string `json:"source"`
	Action  string `json:"action"`
}

func (e ErrorResponse) Error() string {
	if len(e.Status) == 0 {
		e.Status = fmt.Sprintf("Uknown status: %d", e.Code)
	}
	if len(e.Source) == 0 {
		return fmt.Sprintf("(%d) - %s", e.Code, e.Message)
	} else if len(e.Action) == 0 {
		return fmt.Sprintf("(%d)- [%s] - %s", e.Code, e.Source, e.Message)
	} else {
		return fmt.Sprintf("(%d)- [%s-%s] - %s", e.Code, e.Source, e.Action, e.Message)
	}
}

func NewError(statusCode int, message string, sourceAction ...string) *ErrorResponse {
	errResp := &ErrorResponse{
		Code:    statusCode,
		Status:  http.StatusText(statusCode),
		Message: message,
	}

	if len(sourceAction) > 0 {
		errResp.Source = sourceAction[0]
	}
	if len(sourceAction) > 1 {
		errResp.Action = sourceAction[1]
	}

	return errResp
}
