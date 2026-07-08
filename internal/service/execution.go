package service

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/antonio-alexander/go-blog-observability/internal/pkg/errors"
)

func errorTypeToStatusCode(e errors.ErrorType) int {
	switch e {
	default:
		return http.StatusInternalServerError
	case errors.ErrorTypeNotFound, errors.ErrorTypeNotCached:
		return http.StatusNotFound
	case errors.ErrorTypeNotCachedRetry:
		return http.StatusTooManyRequests
	case errors.ErrorTypeConflict:
		return http.StatusConflict
	case errors.ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case errors.ErrorTypeNotImplemented:
		return http.StatusNotImplemented
	case errors.ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case errors.ErrorTypeTooMuchData:
		return http.StatusRequestEntityTooLarge
	}
}

func getAuthorization(req *http.Request) string {
	return req.Header.Get("Authorization")
}

func getCorrelationId(req *http.Request) string {
	if correlationId := req.Header.Get("Correlation-Id"); correlationId != "" {
		return correlationId
	}
	if correlationId := req.URL.Query().Get("correlation_id"); correlationId != "" {
		return correlationId
	}
	return ""
}

func handleResponse(writer http.ResponseWriter, err error, items ...any) error {
	var statusCode int
	var bytes []byte

	if err != nil {
		switch {
		default:
			bytes, err = json.Marshal(errors.Must(errors.New(err, errors.ErrorTypeUnknown)))
			if err != nil {
				return err
			}
			statusCode = http.StatusInternalServerError
		case errors.Is(err, &http.MaxBytesError{}):
			bytes, err = json.Marshal(errors.ErrorCommon{
				ErrorMessage: "too much data",
				ErrorType:    errors.ErrorTypeTooMuchData,
				Local:        true,
			})
			if err != nil {
				return err
			}
			statusCode = http.StatusRequestEntityTooLarge
		case errors.Is(err, errors.ErrNotCached),
			errors.Is(err, errors.ErrNotFound),
			errors.Is(err, errors.ErrUnknown):
			e, ok := errors.AsType[errors.ErrorCommon](err)
			if ok {
				bytes, err = json.Marshal(e)
				statusCode = errorTypeToStatusCode(e.ErrorType)
			} else {
				statusCode = http.StatusInternalServerError
				bytes, err = json.Marshal(&errors.ErrorCommon{
					Err:                err,
					ErrorMessage:       err.Error(),
					ErrorMessageDetail: "",
					ErrorType:          errors.ErrorTypeUnknown,
					Local:              true,
				})
			}
			if err != nil {
				return err
			}
		case errors.Is(err, errors.ErrNotCachedRetry):
			e, _ := errors.AsType[errors.ErrorCommon](err)
			bytes, err = json.Marshal(e)
			if err != nil {
				return err
			}
			statusCode = errorTypeToStatusCode(e.ErrorType)
			writer.Header().Set("Retry-After", "10")
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		writer.WriteHeader(statusCode)
		if _, err := writer.Write(bytes); err != nil {
			return err
		}
		return nil
	}
	switch {
	default:
		bytes, err = json.Marshal(items[0])
		if err != nil {
			return err
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		statusCode = http.StatusOK
	case len(items) <= 0:
		statusCode = http.StatusNoContent
	}
	writer.WriteHeader(statusCode)
	if _, err := writer.Write(bytes); err != nil {
		return err
	}
	return nil
}

func readRequestBody(r *http.Request, v ...any) ([]byte, error) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(bytes) > 0 && len(v) > 0 {
		if err := json.Unmarshal(bytes, v[0]); err != nil {
			return nil, err
		}
	}
	return bytes, nil
}
