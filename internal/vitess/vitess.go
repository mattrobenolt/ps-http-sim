package vitess

import (
	"context"
	"unsafe"

	querypb "github.com/planetscale/vitess-types/gen/vitess/query/v16"
	vtrpcpb "github.com/planetscale/vitess-types/gen/vitess/vtrpc/v16"
	"vitess.io/vitess/go/sqltypes"
	vitessquerypb "vitess.io/vitess/go/vt/proto/query"
)

// Code returns the error code if it's a vtError.
// If err is nil, it returns ok.
func Code(err error) vtrpcpb.Code {
	if err == nil {
		return vtrpcpb.Code_OK
	}

	// Handle some special cases.
	switch err {
	case context.Canceled:
		return vtrpcpb.Code_CANCELED
	case context.DeadlineExceeded:
		return vtrpcpb.Code_DEADLINE_EXCEEDED
	}
	return vtrpcpb.Code_UNKNOWN
}

func ToVTRPC(err error) *vtrpcpb.RPCError {
	if err == nil {
		return nil
	}
	return &vtrpcpb.RPCError{
		Code:    Code(err),
		Message: err.Error(),
	}
}

func ResultToProto(qr *sqltypes.Result) *querypb.QueryResult {
	return unsafeCastQueryResult(sqltypes.ResultToProto3(qr))
}

func castTo[RT any, T any](a T) *RT {
	return (*(**RT)(unsafe.Pointer(&a)))
}

func unsafeCastQueryResult(qr *vitessquerypb.QueryResult) *querypb.QueryResult {
	return castTo[querypb.QueryResult](qr)
}
