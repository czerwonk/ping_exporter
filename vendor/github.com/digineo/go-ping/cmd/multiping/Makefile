TARGET = multiping

include ../common.mk

.PHONY: test
test: all
	go test ./...
	./$(TARGET) golang.org cloudflare.com 2>$(TARGET).log
	cat $(TARGET).log
