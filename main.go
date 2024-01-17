package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/planetscale/log"
	"github.com/planetscale/psdb/auth"
	psdbv1alpha1 "github.com/planetscale/psdb/types/psdb/v1alpha1"
	"github.com/planetscale/psdb/types/psdb/v1alpha1/psdbv1alpha1connect"
	querypb "github.com/planetscale/vitess-types/gen/vitess/query/v16"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"vitess.io/vitess/go/mysql"
	"vitess.io/vitess/go/sqlescape"

	"github.com/mattrobenolt/ps-http-sim/internal/session"
	"github.com/mattrobenolt/ps-http-sim/internal/vitess"
)

var (
	connPool = map[mysqlConnKey]*timedConn{}
	connMu   sync.RWMutex
)

type mysqlConnKey struct {
	username, pass, dbname, session string
}

type timedConn struct {
	*mysql.Conn
	mu    sync.Mutex
	timer *time.Timer
}

var (
	commandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flagListenAddr       = commandLine.String("listen-addr", "127.0.0.1", "HTTP server address")
	flagListenPort       = commandLine.Uint("listen-port", 8080, "HTTP server port")
	flagMySQLAddr        = commandLine.String("mysql-addr", "127.0.0.1", "MySQL address")
	flagMySQLPort        = commandLine.Uint("mysql-port", 3306, "MySQL port")
	flagMySQLNoPass      = commandLine.Bool("mysql-no-pass", false, "Don't use password for MySQL connection")
	flagMySQLIdleTimeout = commandLine.Duration("mysql-idle-timeout", 10*time.Second, "MySQL connection idle timeout")
	flagMySQLMaxRows     = commandLine.Uint("mysql-max-rows", 1000, "Max rows for a single query result set")
	flagMySQLDbname      = commandLine.String("mysql-dbname", "mysql", "MySQL database to connect to")
)

var errSessionInUse = errors.New("session already in use")

// getConn gets or dials a connection from the connection pool
// connections are maintained unique for credential combos and session id
// since this isn't meant to truly represent reality, it's possible you
// can do things with connections locally by munging session ids or auth
// that aren't allowed on PlanetScale. This is meant to just mimic the public API.
func getConn(ctx context.Context, uname, pass, dbname, session string) (*timedConn, error) {
	key := mysqlConnKey{uname, pass, dbname, session}

	// check first if there's already a connection
	connMu.RLock()
	if conn, ok := connPool[key]; ok {
		defer connMu.RUnlock()

		if conn.mu.TryLock() {
			if conn.timer == nil {
				conn.timer = time.AfterFunc(*flagMySQLIdleTimeout, func() {
					closeIdleConn(uname, pass, dbname, session)
				})
			} else {
				conn.timer.Reset(*flagMySQLIdleTimeout)
			}
			return conn, nil
		} else {
			return nil, errSessionInUse
		}
	}
	connMu.RUnlock()

	// if not, dial for a new connection
	// without a lock, so parallel dials can happen
	rawConn, err := dial(ctx, uname, pass, dbname)
	if err != nil {
		return nil, err
	}

	// lock to write to map
	connMu.Lock()
	connPool[key] = &timedConn{Conn: rawConn}
	connMu.Unlock()

	// since it was parallel, the last one would have won and been written
	// so re-read back so we use the conn that was actually stored in the pool
	return getConn(ctx, uname, pass, dbname, session)
}

func returnConn(conn *timedConn) {
	conn.timer.Reset(*flagMySQLIdleTimeout)
	conn.mu.Unlock()
}

func closeIdleConn(uname, pass, dbname, session string) {
	logger.Debug("closing idle connection",
		log.String("username", uname),
		log.String("session_id", session),
	)
	closeConn(uname, pass, dbname, session)
}

func closeConn(uname, pass, dbname, session string) {
	key := mysqlConnKey{uname, pass, dbname, session}

	connMu.Lock()
	if conn, ok := connPool[key]; ok {
		conn.Close()
		if conn.timer != nil {
			conn.timer.Stop()
		}
		delete(connPool, key)
	}
	connMu.Unlock()
}

// dial connects to the underlying MySQL server, and switches to the underlying
// database automatically.
func dial(ctx context.Context, uname, pass, dbname string) (*mysql.Conn, error) {
	if *flagMySQLNoPass {
		pass = ""
	}
	conn, err := mysql.Connect(ctx, &mysql.ConnParams{
		Host:  *flagMySQLAddr,
		Port:  int(*flagMySQLPort),
		Uname: uname,
		Pass:  pass,
	})
	if err != nil {
		return nil, err
	}
	if dbname != "" {
		if _, err := conn.ExecuteFetch("USE "+sqlescape.EscapeID(dbname), 1, false); err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

func init() {
	// Vitess doesn't play nicely, so replace the entire default flagset
	flag.CommandLine = commandLine
	flag.Parse()
}

var logger *log.Logger

func main() {
	cfg := log.NewPlanetScaleConfig("pretty", log.DebugLevel)
	logger, _ = cfg.Build()
	defer logger.Sync()

	mux := http.NewServeMux()
	mux.Handle(psdbv1alpha1connect.NewDatabaseHandler(server{}))

	logger.Info("running",
		log.String("addr", *flagListenAddr),
		log.Uint("port", *flagListenPort),
	)
	panic(http.ListenAndServe(
		fmt.Sprintf("%s:%d", *flagListenAddr, *flagListenPort),
		h2c.NewHandler(mux, &http2.Server{}),
	))
}

type server struct{}

func (server) CreateSession(
	ctx context.Context,
	req *connect.Request[psdbv1alpha1.CreateSessionRequest],
) (*connect.Response[psdbv1alpha1.CreateSessionResponse], error) {
	ll := logger.With(
		log.String("method", "CreateSession"),
		log.String("content_type", req.Header().Get("Content-Type")),
	)

	creds, err := auth.ParseWithSecret(req.Header().Get("Authorization"))
	if err != nil || creds.Type() != auth.BasicAuthType {
		ll.Error("unauthenticated", log.Error(err))
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	ll = ll.With(
		log.String("user", creds.Username()),
	)

	sess := session.New(*flagMySQLDbname)
	sessionID := session.UUID(sess)
	dbname := session.DBName(sess)

	conn, err := getConn(ctx, creds.Username(), string(creds.SecretBytes()), dbname, sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "Access denied for user") {
			ll.Error("unauthenticated", log.Error(err))
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		} else if err == errSessionInUse {
			ll.Warn(err.Error())
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				fmt.Errorf("%s: %s", err.Error(), sessionID),
			)
		}
		ll.Error("failed to connect", log.Error(err))
		return nil, err
	}
	defer returnConn(conn)

	ll.Info("ok")

	return connect.NewResponse(&psdbv1alpha1.CreateSessionResponse{
		Branch: gonanoid.Must(), // there is no branch, so fake it
		User: &psdbv1alpha1.User{
			Username: creds.Username(),
			Psid:     "planetscale-1",
		},
		Session: sess,
	}), nil
}

func (server) Execute(
	ctx context.Context,
	req *connect.Request[psdbv1alpha1.ExecuteRequest],
) (*connect.Response[psdbv1alpha1.ExecuteResponse], error) {
	ll := logger.With(
		log.String("method", "Execute"),
		log.String("content_type", req.Header().Get("Content-Type")),
	)

	creds, err := auth.ParseWithSecret(req.Header().Get("Authorization"))
	if err != nil || creds.Type() != auth.BasicAuthType {
		ll.Error("unauthenticated", log.Error(err))
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	ll = ll.With(
		log.String("user", creds.Username()),
	)

	msg := req.Msg
	query := msg.Query
	sess := msg.Session
	clientSession := sess != nil

	// if there is no session, let's generate a new one
	if !clientSession {
		sess = session.New(*flagMySQLDbname)
	}
	sessionID := session.UUID(sess)
	dbname := session.DBName(sess)

	ll = ll.With(
		log.String("query", query),
		log.String("session_id", sessionID),
		log.Bool("client_session", clientSession),
	)

	conn, err := getConn(ctx, creds.Username(), string(creds.SecretBytes()), dbname, sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "Access denied for user") {
			ll.Error("unauthenticated", log.Error(err))
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		} else if err == errSessionInUse {
			ll.Warn(err.Error())
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				fmt.Errorf("%s: %s", err.Error(), sessionID),
			)
		}
		ll.Error("failed to connect", log.Error(err))
		return nil, err
	}
	defer returnConn(conn)

	ll.Info("ok")

	start := time.Now()
	// This is a gross simplificiation, but is likely sufficient
	qr, err := conn.ExecuteFetch(query, int(*flagMySQLMaxRows), true)
	timing := time.Since(start)
	session.Update(qr, sess)

	return connect.NewResponse(&psdbv1alpha1.ExecuteResponse{
		Session: sess,
		Result:  vitess.ResultToProto(qr),
		Error:   vitess.ToVTRPC(err),
		Timing:  timing.Seconds(),
	}), nil
}

func (server) ExecuteBatch(
	ctx context.Context,
	req *connect.Request[psdbv1alpha1.ExecuteBatchRequest],
) (*connect.Response[psdbv1alpha1.ExecuteBatchResponse], error) {
	ll := logger.With(
		log.String("method", "ExecuteBatch"),
		log.String("content_type", req.Header().Get("Content-Type")),
	)

	creds, err := auth.ParseWithSecret(req.Header().Get("Authorization"))
	if err != nil || creds.Type() != auth.BasicAuthType {
		ll.Error("unauthenticated", log.Error(err))
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	ll = ll.With(
		log.String("user", creds.Username()),
	)

	msg := req.Msg
	queries := msg.Queries
	sess := msg.Session
	clientSession := sess != nil

	// if there is no session, let's generate a new one
	if !clientSession {
		sess = session.New(*flagMySQLDbname)
	}
	sessionID := session.UUID(sess)
	dbname := session.DBName(sess)

	ll = ll.With(
		log.Strings("queries", queries),
		log.String("session_id", sessionID),
		log.Bool("client_session", clientSession),
	)

	conn, err := getConn(ctx, creds.Username(), string(creds.SecretBytes()), dbname, sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "Access denied for user") {
			ll.Error("unauthenticated", log.Error(err))
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		} else if err == errSessionInUse {
			ll.Warn(err.Error())
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				fmt.Errorf("%s: %s", err.Error(), sessionID),
			)
		}
		ll.Error("failed to connect", log.Error(err))
		return nil, err
	}
	defer returnConn(conn)

	ll.Info("ok")

	results := make([]*querypb.ResultWithError, 0, len(queries))
	for _, query := range queries {
		// This is a gross simplificiation, but is likely sufficient
		qr, err := conn.ExecuteFetch(query, int(*flagMySQLMaxRows), true)
		session.Update(qr, sess)
		results = append(results, &querypb.ResultWithError{
			Result: vitess.ResultToProto(qr),
			Error:  vitess.ToVTRPC(err),
		})
	}

	return connect.NewResponse(&psdbv1alpha1.ExecuteBatchResponse{
		Session: sess,
		Results: results,
	}), nil
}

func (server) StreamExecute(
	ctx context.Context,
	req *connect.Request[psdbv1alpha1.ExecuteRequest],
	stream *connect.ServerStream[psdbv1alpha1.ExecuteResponse],
) error {
	ll := logger.With(
		log.String("method", "StreamExecute"),
		log.String("content_type", req.Header().Get("Content-Type")),
	)

	creds, err := auth.ParseWithSecret(req.Header().Get("Authorization"))
	if err != nil || creds.Type() != auth.BasicAuthType {
		ll.Error("unauthenticated", log.Error(err))
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	ll = ll.With(
		log.String("user", creds.Username()),
	)

	msg := req.Msg
	query := msg.Query
	sess := msg.Session
	clientSession := sess != nil

	// if there is no session, let's generate a new one
	if !clientSession {
		sess = session.New(*flagMySQLDbname)
	}
	sessionID := session.UUID(sess)
	dbname := session.DBName(sess)

	ll = ll.With(
		log.String("query", query),
		log.String("session_id", sessionID),
		log.Bool("client_session", clientSession),
	)

	conn, err := getConn(ctx, creds.Username(), string(creds.SecretBytes()), dbname, sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "Access denied for user") {
			ll.Error("unauthenticated", log.Error(err))
			return connect.NewError(connect.CodeUnauthenticated, err)
		} else if err == errSessionInUse {
			ll.Warn(err.Error())
			return connect.NewError(
				connect.CodePermissionDenied,
				fmt.Errorf("%s: %s", err.Error(), sessionID),
			)
		}
		ll.Error("failed to connect", log.Error(err))
		return err
	}
	defer returnConn(conn)

	// fake a streaming response by just returning 2 messages of the same payload
	// far from reality, but a simple way to exercise the protocol.
	qr, err := conn.ExecuteFetch(query, int(*flagMySQLMaxRows), true)
	session.Update(qr, sess)

	ll.Info("send msg")
	if err := stream.Send(&psdbv1alpha1.ExecuteResponse{
		Session: sess,
		Result:  vitess.ResultToProto(qr),
		Error:   vitess.ToVTRPC(err),
	}); err != nil {
		ll.Error("send failed", log.Error(err))
		return err
	}

	ll.Info("send msg")
	if err := stream.Send(&psdbv1alpha1.ExecuteResponse{
		Session: sess,
		Result:  vitess.ResultToProto(qr),
		Error:   vitess.ToVTRPC(err),
	}); err != nil {
		ll.Error("send failed", log.Error(err))
		return err
	}

	return nil
}

func (server) CloseSession(
	ctx context.Context,
	req *connect.Request[psdbv1alpha1.CloseSessionRequest],
) (*connect.Response[psdbv1alpha1.CloseSessionResponse], error) {
	ll := logger.With(
		log.String("method", "CloseSession"),
		log.String("content_type", req.Header().Get("Content-Type")),
	)

	creds, err := auth.ParseWithSecret(req.Header().Get("Authorization"))
	if err != nil || creds.Type() != auth.BasicAuthType {
		ll.Error("unauthenticated", log.Error(err))
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	sess := req.Msg.Session
	if sess != nil {
		closeConn(creds.Username(), string(creds.SecretBytes()), session.DBName(sess), session.UUID(sess))
	}

	return connect.NewResponse(&psdbv1alpha1.CloseSessionResponse{
		Session: session.Reset(sess),
	}), nil
}

func (server) Prepare(
	ctx context.Context,
	req *connect.Request[psdbv1alpha1.PrepareRequest],
) (*connect.Response[psdbv1alpha1.PrepareResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}
