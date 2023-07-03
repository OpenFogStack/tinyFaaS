.PHONY: all

all: api_pb2.py api_pb2.pyi api_pb2_grpc.py api.pb.go api_grpc.pb.go

# requires protoc-gen-go and protoc-gen-go-grpc
api.pb.go api_grpc.pb.go: api.proto
	@protoc -I . $< --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative

# requires grpcio-tools and mypy-protobuf
api_pb2.py api_pb2.pyi api_pb2_grpc.py: api.proto
	@python3 -m grpc_tools.protoc -I . --python_out=. --grpc_python_out=. --mypy_out=. $<
