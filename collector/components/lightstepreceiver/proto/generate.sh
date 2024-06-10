#!/usr/bin/env bash

# Run this in the ../ i.e., lightstepreceiver.
rm -rf internal
mkdir internal

protoc -I proto --go_out=internal --go-grpc_out=internal proto/collector.proto

mv internal/github.com/lightstep/sn-collector/collector/lightstepreceiver/internal/collectorpb internal
rm -rf internal/github.com

mdatagen metadata.yaml
