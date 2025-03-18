CXX = g++
CXXFLAGS = -Wall -std=c++17 -I/Users/jeff/go/src/github.com/accelera-official/systemc/include
LDFLAGS = -L/Users/jeff/go/src/github.com/accelera-official/systemc/lib-macosx64 -lsystemc
SOURCES = sc3.cpp
TARGET = sc3

all: $(TARGET)

$(TARGET): $(SOURCES)
	$(CXX) $(CXXFLAGS) $(SOURCES) $(LDFLAGS) -o $(TARGET)

clean:
	rm -f *.o $(TARGET)
