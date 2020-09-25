package service

type RpcOptions struct {
	Host       string
	Port       int16
	Credential interface{}
}

type TlsFileCredential struct {
	File string
}

type UsernamePasswordCredential struct {
	Username string
	Password string
}

type MacaroonFileCredential struct {
	File string
}

type RpcProvider interface {
	ConfigureRpc(options *RpcOptions)
}
