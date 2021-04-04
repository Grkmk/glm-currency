.PHONY: protos

protos:
	protoc -I protos/ --go-grpc_out=protos --go_out=protos protos/currency.proto