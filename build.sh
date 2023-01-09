#!/usr/bin/bash

# Linux
env GOOS=linux GOARCH=amd64 go build -tags netgo -o tikmeh 
echo "Linux amd64 done."

# Windows
env GOOS=windows GOARCH=amd64 go build -tags netgo -o tikmeh.exe
echo "Windows amd64 done."

# MacOS
env GOOS=darwin GOARCH=amd64 go build -tags netgo -o tikmeh_macos
echo "MacOS amd64 done."
env GOOS=darwin GOARCH=arm64 go build -tags netgo -o tikmeh_macos_arm64
echo "MacOS arm64 done."
