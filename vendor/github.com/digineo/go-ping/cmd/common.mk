.PHONY: all
all: $(TARGET)

$(TARGET):
	go build -o $@

.PHONY: clean
clean:
	rm -f $(TARGET) $(TARGET).log
