.PHONY: all
all: $(TARGET)

$(TARGET):
	go build -o $@
	$(MAKE) setcap

.PHONY: setcap
setcap: $(TARGET)
	sudo setcap cap_net_raw+ep $<

.PHONY: clean
clean:
	rm -f $(TARGET) $(TARGET).log
