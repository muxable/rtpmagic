#!/bin/bash

docker run -v $PWD:/defs -w /defs rvolosatovs/protoc --go_out=. --dart_out=. --proto_path=/defs muxer.proto