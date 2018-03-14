TARGET = ping-test

include ../common.mk

.PHONY: test
test: all
	@truncate -s0 $(TARGET).log
	@./$(TARGET) -4 golang.org      2>>$(TARGET).log
	@./$(TARGET) -6 golang.org      2>>$(TARGET).log
	@./$(TARGET) -4 cloudflare.com  2>>$(TARGET).log
	@./$(TARGET) -6 cloudflare.com  2>>$(TARGET).log

	@echo error log:
	@cat $(TARGET).log
