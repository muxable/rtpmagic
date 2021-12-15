#!/bin/bash

sudo GST_DEBUG=3 /usr/local/go/bin/go run cmd/muxer/main.go -video-src "uridecodebin uri=\"rtmp://localhost/live/mugit\"" -audio-src "uridecodebin uri=\"rtmp://localhost/live/mugit\"" -dest 34.86.30.237:5000
