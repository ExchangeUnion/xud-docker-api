package rpc

type Conn interface {
	GetClient() interface{}
	Open()
	Close() error
}
