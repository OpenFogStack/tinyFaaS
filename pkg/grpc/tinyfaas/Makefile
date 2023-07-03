.PHONY: all

PROTO = tinyfaas

all: ${PROTO}_pb2.py ${PROTO}_pb2.pyi ${PROTO}_pb2_grpc.py ${PROTO}.pb.go ${PROTO}_grpc.pb.go

# requires protoc,  protoc-gen-go and protoc-gen-go-grpc
# install from your package manager, e.g.:
# 	brew install protobuf
# 	brew install protoc-gen-go
#	brew install protoc-gen-go-grpc
${PROTO}.pb.go ${PROTO}_grpc.pb.go: ${PROTO}.proto
	@protoc -I . $< --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=require_unimplemented_servers=false,paths=source_relative

# requires grpcio-tools and mypy-protobuf
# 	python3 -m pip install -r requirements.txt
${PROTO}_pb2.py ${PROTO}_pb2.pyi ${PROTO}_pb2_grpc.py: ${PROTO}.proto
	@python3 -m grpc_tools.protoc -I . --python_out=. --grpc_python_out=. --mypy_out=. $<
