package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

// MacaroonCredential implements the credentials.PerRPCCredentials interface.
type MacaroonCredential string

// GetRequestMetadata implements the PerRPCCredentials interface. This method
// is required in order to pass the wrapped macaroon into the gRPC context.
// With this, the macaroon will be available within the request handling scope
// of the ultimate gRPC server implementation.
func (t MacaroonCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	data, err := ioutil.ReadFile(string(t))
	if err != nil {
		return nil, err
	}
	md := make(map[string]string)
	md["macaroon"] = hex.EncodeToString(data)
	return md, nil
}

// RequireTransportSecurity implements the PerRPCCredentials interface.
func (t MacaroonCredential) RequireTransportSecurity() bool {
	return true
}

type GrpcConn struct {
	host string
	port uint16
	tlsCert string
	macaroon string

	conn *grpc.ClientConn
	mutex *sync.RWMutex
	client interface{}

	newClientFunc func(*grpc.ClientConn) interface{}

	logger *logrus.Entry
}

func NewGrpcConn(host string, port uint16, tlsCert string, macaroon string, logger *logrus.Entry, newClientFunc func(*grpc.ClientConn) interface{}) *GrpcConn {
	conn := &GrpcConn{
		host: host,
		port: port,
		tlsCert: tlsCert,
		macaroon: macaroon,

		conn: nil,
		mutex: &sync.RWMutex{},
		client: nil,

		newClientFunc: newClientFunc,

		logger: logger,
	}

	return conn
}

func (t *GrpcConn) Update(host string, port uint16, tlsCert string, macaroon string) error {
	// TODO update when tlsCert or macaroon file changed
	t.host = host
	t.port = port
	t.tlsCert = tlsCert
	t.macaroon = macaroon

	if err := t.reopen(); err != nil {
		return err
	}

	return nil
}

func (t *GrpcConn) reopen() error {
	if err := t.Close(); err != nil {
		return err
	}
	t.Open()
	return nil
}

func (t *GrpcConn) Open() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := t.connect(ctx)
		cancel()
		if err == nil {
			break
		}
		t.logger.Debugf("Failed to establish gRPC connection: %s", err)
		time.Sleep(3 * time.Second)
	}
}

func (t *GrpcConn) connect(ctx context.Context) error {
	creds, err := credentials.NewClientTLSFromFile(t.tlsCert, "localhost")
	if err != nil {
		return err
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(creds))
	opts = append(opts, grpc.WithBlock())

	if t.macaroon != "" {
		if _, err := os.Stat(t.macaroon); os.IsNotExist(err) {
			return err
		}
		macaroonCred := MacaroonCredential(t.macaroon)
		opts = append(opts, grpc.WithPerRPCCredentials(&macaroonCred))
	}

	addr := fmt.Sprintf("%s:%d", t.host, t.port)

	t.logger.Debugf("Establishing gRPC connection to %s (tlsCert=%s, macaroon=%s)", addr, t.tlsCert, t.macaroon)
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return err
	}
	t.logger.Debugf("Established gRPC connection")

	go func() {
		changed := conn.WaitForStateChange(ctx, connectivity.Ready)
		if changed {
			t.logger.Debugf("gRPC connection broken: %s", conn.GetState())
		}
	}()

	t.mutex.Lock()
	t.conn = conn
	t.client = t.newClientFunc(conn)
	t.mutex.Unlock()

	return nil
}

func (t *GrpcConn) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.conn != nil {
		err := t.conn.Close()
		if err != nil {
			return err
		}
		t.conn = nil
		t.client = nil
	}
	return nil
}

func (t *GrpcConn) GetClient() interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.client
}