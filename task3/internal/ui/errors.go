package ui

import (
	"context"
	"errors"
)

func operationErrorText(err error) string {
	switch {
	case err == nil:
		return ""

	case errors.Is(err, context.DeadlineExceeded):
		return "Превышено время ожидания. Попробуйте ещё раз."

	case errors.Is(err, context.Canceled):
		return "Операция отменена."

	default:
		return err.Error()
	}
}
