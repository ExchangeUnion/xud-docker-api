package service

type RpcOptions struct {
	Host       string
	Port       int16
	Credential interface{}
}

type TlsFileCredential struct {
}

type UsernamePasswordCredential struct {
	Username string
	Password string
}

type MacaroonFileCredential struct {
}

type RpcProvider interface {
	ConfigureRpc(options *RpcOptions)
}
