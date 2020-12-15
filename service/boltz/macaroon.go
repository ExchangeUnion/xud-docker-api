package boltz

import (
	"context"
	"encoding/hex"
	"io/ioutil"
)

// MacaroonCredential implements the credentials.PerRPCCredentials interface.
type MacaroonCredential struct {
	Admin string
}

// GetRequestMetadata implements the PerRPCCredentials interface. This method
// is required in order to pass the wrapped macaroon into the gRPC context.
// With this, the macaroon will be available within the request handling scope
// of the ultimate gRPC server implementation.
func (t *MacaroonCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	data, err := ioutil.ReadFile(t.Admin)
	if err != nil {
		return nil, err
	}
	md := make(map[string]string)
	md["macaroon"] = hex.EncodeToString(data)
	return md, nil
}

// RequireTransportSecurity implements the PerRPCCredentials interface.
func (t *MacaroonCredential) RequireTransportSecurity() bool {
	return true
}
