package errcode

import (
	"encoding/json"
	"net/http"
	"strconv"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrCode struct {
	Code       int    `json:"code"`
	HttpStatus int    `json:"http_status"`
	Msg        string `json:"msg"`
}

func (e *ErrCode) Error() string {
	return e.Msg
}

func (e *ErrCode) GRPCCode() codes.Code {
	switch e.Code {
	case ErrInvalidParams.Code:
		return codes.InvalidArgument
	case ErrUnauthorized.Code, ErrInvalidCredential.Code:
		return codes.Unauthenticated
	case ErrForbidden.Code, ErrNotPurchased.Code:
		return codes.PermissionDenied
	case ErrUsernameExists.Code, ErrAlreadyPurchased.Code:
		return codes.AlreadyExists
	case ErrCourseNotFound.Code, ErrVideoNotFound.Code:
		return codes.NotFound
	case ErrOrderProcessing.Code:
		return codes.Aborted
	case ErrInternal.Code, ErrOSS.Code:
		return codes.Internal
	default:
		return codes.Unknown
	}
}

// ToGRPCError converts business errors to semantic gRPC codes and carries
// business metadata in ErrorInfo for cross-service callers.
func (e *ErrCode) ToGRPCError() error {
	st := status.New(e.GRPCCode(), e.Msg)
	stWithDetails, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "BIZ_ERROR",
		Domain: "geekedu",
		Metadata: map[string]string{
			"code":        strconv.Itoa(e.Code),
			"http_status": strconv.Itoa(e.HttpStatus),
			"msg":         e.Msg,
		},
	})
	if err == nil {
		return stWithDetails.Err()
	}

	// Fallback for unexpected detail-packing failures and older clients.
	b, _ := json.Marshal(e)
	return status.Error(e.GRPCCode(), string(b))
}

// FromGRPCError extracts business error metadata from gRPC errors.
func FromGRPCError(err error) *ErrCode {
	st, ok := status.FromError(err)
	if !ok {
		return ErrInternal
	}

	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.Domain != "geekedu" || info.Reason != "BIZ_ERROR" {
			continue
		}
		code, codeErr := strconv.Atoi(info.Metadata["code"])
		httpStatus, statusErr := strconv.Atoi(info.Metadata["http_status"])
		msg := info.Metadata["msg"]
		if codeErr == nil && statusErr == nil && code != 0 && msg != "" {
			return &ErrCode{Code: code, HttpStatus: httpStatus, Msg: msg}
		}
	}

	// Backward compatibility: older services encoded ErrCode as JSON in message.
	var bizErr ErrCode
	if e := json.Unmarshal([]byte(st.Message()), &bizErr); e == nil && bizErr.Code != 0 {
		return &bizErr
	}

	return FromGRPCCode(st.Code(), st.Message())
}

func FromGRPCCode(code codes.Code, msg string) *ErrCode {
	switch code {
	case codes.InvalidArgument:
		return ErrInvalidParams
	case codes.Unauthenticated:
		return ErrUnauthorized
	case codes.PermissionDenied:
		return ErrForbidden
	case codes.NotFound:
		return ErrCourseNotFound
	case codes.AlreadyExists:
		return &ErrCode{Code: ErrAlreadyPurchased.Code, HttpStatus: ErrAlreadyPurchased.HttpStatus, Msg: msg}
	case codes.Aborted:
		return ErrOrderProcessing
	case codes.Internal:
		return ErrInternal
	default:
		return ErrInternal
	}
}

var (
	Success              = &ErrCode{Code: 0, HttpStatus: http.StatusOK, Msg: "success"}
	ErrInvalidParams     = &ErrCode{Code: 10001, HttpStatus: http.StatusBadRequest, Msg: "invalid params"}
	ErrUnauthorized      = &ErrCode{Code: 10002, HttpStatus: http.StatusUnauthorized, Msg: "unauthorized"}
	ErrForbidden         = &ErrCode{Code: 10003, HttpStatus: http.StatusForbidden, Msg: "forbidden"}
	ErrUsernameExists    = &ErrCode{Code: 20001, HttpStatus: http.StatusConflict, Msg: "username already exists"}
	ErrInvalidCredential = &ErrCode{Code: 20002, HttpStatus: http.StatusUnauthorized, Msg: "invalid credentials"}
	ErrCourseNotFound    = &ErrCode{Code: 30001, HttpStatus: http.StatusNotFound, Msg: "course not found"}
	ErrVideoNotFound     = &ErrCode{Code: 30002, HttpStatus: http.StatusNotFound, Msg: "video not found"}
	ErrNotPurchased      = &ErrCode{Code: 30003, HttpStatus: http.StatusForbidden, Msg: "course not purchased"}
	ErrAlreadyPurchased  = &ErrCode{Code: 40001, HttpStatus: http.StatusConflict, Msg: "already purchased"}
	ErrOrderProcessing   = &ErrCode{Code: 40002, HttpStatus: http.StatusConflict, Msg: "order processed,do not submit."}
	ErrInternal          = &ErrCode{Code: 50001, HttpStatus: http.StatusInternalServerError, Msg: "internal error"}
	ErrOSS               = &ErrCode{Code: 50002, HttpStatus: http.StatusInternalServerError, Msg: "oss error"}
)
