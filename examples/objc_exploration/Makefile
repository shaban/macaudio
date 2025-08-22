# Objective-C AVAudioEngine TimePitch Exploration

CC = clang
CFLAGS = -fobjc-arc -framework AVFoundation -framework Foundation
TARGET = timepitch_exploration
SOURCE = main.m

all: $(TARGET)

$(TARGET): $(SOURCE)
	$(CC) $(CFLAGS) -o $(TARGET) $(SOURCE)

clean:
	rm -f $(TARGET)

run: $(TARGET)
	./$(TARGET)

.PHONY: all clean run
