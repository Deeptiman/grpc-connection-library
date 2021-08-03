#!/bin/bash
protoc -I C:/Users/asusd/go/pkg/mod/protobuf/src --proto_path . --go_out=plugins=grpc:. ping.proto