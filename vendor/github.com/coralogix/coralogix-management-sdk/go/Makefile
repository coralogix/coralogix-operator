# Copyright 2024 Coralogix Ltd.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     https://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TEST?=$$(go list ./... | grep -v 'vendor')
NAME=cxsdk
BINARY=${NAME}
VERSION=0.1
OS_ARCH=darwin_arm64
BUILD_ARGS=-ldflags "-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn"
default: build

.PHONY: proto-clean
proto-clean: ## Clean generated proto.
	@rm -rf internal/*

.PHONY: proto-compile
proto-compile: ## Compile message protobuf and gRPC service files.
	@./scripts/generate_go_protos.sh

build: proto-clean proto-compile build-source-files

build-source-files:
	go build -o out/${BINARY}.so ${BUILD_ARGS}

release:
	GOOS=darwin GOARCH=amd64 go build -o out/${BINARY}_${VERSION}_darwin_amd64.dylib
	GOOS=freebsd GOARCH=386 go build -o out/${BINARY}_${VERSION}_freebsd_386.so
	GOOS=freebsd GOARCH=amd64 go build -o out/${BINARY}_${VERSION}_freebsd_amd64.so
	GOOS=freebsd GOARCH=arm go build -o out/${BINARY}_${VERSION}_freebsd_arm.so
	GOOS=linux GOARCH=386 go build -o out/${BINARY}_${VERSION}_linux_386.so
	GOOS=linux GOARCH=amd64 go build -o out/${BINARY}_${VERSION}_linux_amd64.so
	GOOS=linux GOARCH=arm go build -o out/${BINARY}_${VERSION}_linux_arm.so
	GOOS=openbsd GOARCH=386 go build -o out/${BINARY}_${VERSION}_openbsd_386.so
	GOOS=openbsd GOARCH=amd64 go build -o out/${BINARY}_${VERSION}_openbsd_amd64.so
	GOOS=solaris GOARCH=amd64 go build -o out/${BINARY}_${VERSION}_solaris_amd64.so
	GOOS=windows GOARCH=386 go build -o out/${BINARY}_${VERSION}_windows_386.dll
	GOOS=windows GOARCH=amd64 go build -o out/${BINARY}_${VERSION}_windows_amd64.dll

test: build
	go test ${BUILD_ARGS} -v ${TESTARGS} -cover -timeout=120s -parallel=4 ./... 

generate:
	go generate