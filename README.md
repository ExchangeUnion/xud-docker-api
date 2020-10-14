proxy
=====

### Compile gRPC proto files

```bash
protoc -I service/xud/proto xudrpc.proto --go_out=plugins=grpc:service/xud/xudrpc
protoc -I service/lnd/proto rpc.proto --go_out=plugins=grpc:service/lnd/lnrpc
```

### Run in xud-docker environment

```bash
scripts/run.sh testnet
```
