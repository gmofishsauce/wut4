CXX = g++
CXXFLAGS = -Wall -std=c++17 -I/Users/jeff/go/src/github.com/accelera-official/systemc/include
LDFLAGS = -L/Users/jeff/go/src/github.com/accelera-official/systemc/lib-macosx64 -lsystemc
TARGET = scsim

all: $(TARGET)

$(TARGET): scsim.cpp LS283.cpp
	$(CXX) $(CXXFLAGS) scsim.cpp LS283.cpp -o $(TARGET) $(LDFLAGS)

clean:
	rm -f $(TARGET)
