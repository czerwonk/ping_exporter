TARGET = pingnet

include ../common.mk

.PHONY: test
test: all
	./$(TARGET) -c 1 -w 100ms -f 127.0.0.1/24
