// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: psdb/v1alpha1/database.proto

package psdbv1alpha1connect

import (
	context "context"
	errors "errors"
	connect_go "github.com/bufbuild/connect-go"
	v1alpha1 "github.com/mattrobenolt/ps-http-sim/types/psdb/v1alpha1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// DatabaseName is the fully-qualified name of the Database service.
	DatabaseName = "psdb.v1alpha1.Database"
)

// DatabaseClient is a client for the psdb.v1alpha1.Database service.
type DatabaseClient interface {
	CreateSession(context.Context, *connect_go.Request[v1alpha1.CreateSessionRequest]) (*connect_go.Response[v1alpha1.CreateSessionResponse], error)
	Execute(context.Context, *connect_go.Request[v1alpha1.ExecuteRequest]) (*connect_go.Response[v1alpha1.ExecuteResponse], error)
}

// NewDatabaseClient constructs a client for the psdb.v1alpha1.Database service. By default, it uses
// the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewDatabaseClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) DatabaseClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &databaseClient{
		createSession: connect_go.NewClient[v1alpha1.CreateSessionRequest, v1alpha1.CreateSessionResponse](
			httpClient,
			baseURL+"/psdb.v1alpha1.Database/CreateSession",
			opts...,
		),
		execute: connect_go.NewClient[v1alpha1.ExecuteRequest, v1alpha1.ExecuteResponse](
			httpClient,
			baseURL+"/psdb.v1alpha1.Database/Execute",
			opts...,
		),
	}
}

// databaseClient implements DatabaseClient.
type databaseClient struct {
	createSession *connect_go.Client[v1alpha1.CreateSessionRequest, v1alpha1.CreateSessionResponse]
	execute       *connect_go.Client[v1alpha1.ExecuteRequest, v1alpha1.ExecuteResponse]
}

// CreateSession calls psdb.v1alpha1.Database.CreateSession.
func (c *databaseClient) CreateSession(ctx context.Context, req *connect_go.Request[v1alpha1.CreateSessionRequest]) (*connect_go.Response[v1alpha1.CreateSessionResponse], error) {
	return c.createSession.CallUnary(ctx, req)
}

// Execute calls psdb.v1alpha1.Database.Execute.
func (c *databaseClient) Execute(ctx context.Context, req *connect_go.Request[v1alpha1.ExecuteRequest]) (*connect_go.Response[v1alpha1.ExecuteResponse], error) {
	return c.execute.CallUnary(ctx, req)
}

// DatabaseHandler is an implementation of the psdb.v1alpha1.Database service.
type DatabaseHandler interface {
	CreateSession(context.Context, *connect_go.Request[v1alpha1.CreateSessionRequest]) (*connect_go.Response[v1alpha1.CreateSessionResponse], error)
	Execute(context.Context, *connect_go.Request[v1alpha1.ExecuteRequest]) (*connect_go.Response[v1alpha1.ExecuteResponse], error)
}

// NewDatabaseHandler builds an HTTP handler from the service implementation. It returns the path on
// which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewDatabaseHandler(svc DatabaseHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/psdb.v1alpha1.Database/CreateSession", connect_go.NewUnaryHandler(
		"/psdb.v1alpha1.Database/CreateSession",
		svc.CreateSession,
		opts...,
	))
	mux.Handle("/psdb.v1alpha1.Database/Execute", connect_go.NewUnaryHandler(
		"/psdb.v1alpha1.Database/Execute",
		svc.Execute,
		opts...,
	))
	return "/psdb.v1alpha1.Database/", mux
}

// UnimplementedDatabaseHandler returns CodeUnimplemented from all methods.
type UnimplementedDatabaseHandler struct{}

func (UnimplementedDatabaseHandler) CreateSession(context.Context, *connect_go.Request[v1alpha1.CreateSessionRequest]) (*connect_go.Response[v1alpha1.CreateSessionResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("psdb.v1alpha1.Database.CreateSession is not implemented"))
}

func (UnimplementedDatabaseHandler) Execute(context.Context, *connect_go.Request[v1alpha1.ExecuteRequest]) (*connect_go.Response[v1alpha1.ExecuteResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("psdb.v1alpha1.Database.Execute is not implemented"))
}
