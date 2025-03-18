# Makefile for Verilator AND gate simulation

# Project name
TOP = and_gate

# Verilator command
VERILATOR = verilator

# Verilator flags
VERILATOR_FLAGS = -Wall --trace --cc --exe --top-module $(TOP) -Mdir obj_dir

# Source files
VSRCS = and_gate.sv
CPPSRCS = test_and_gate.cpp

# Object files
VERILATOR_OBJS = obj_dir/V$(TOP).o obj_dir/V$(TOP)__Syms.o obj_dir/V$(TOP)__Trace.o obj_dir/V$(TOP)__Trace__Slow.o obj_dir/V$(TOP)__ALL.a
CPPOBJS = test_and_gate.o
OBJS = $(VERILATOR_OBJS) $(CPPOBJS)

# Executable name
EXEC = sim_and_gate

# Default target
all: $(EXEC)

# Rule to compile SystemVerilog files
obj_dir/V$(TOP).o: $(VSRCS)
	$(VERILATOR) $(VERILATOR_FLAGS) $(VSRCS)

# Rule to compile C++ files
%.o: %.cpp
	g++ -I. -Iobj_dir -c $< -o $@ -std=c++17

# Link the executable
$(EXEC): $(OBJS)
	g++ -L. $(CPPOBJS) obj_dir/V$(TOP)__ALL.a -o $@ -lstdc++ -std=c++17 -lm

# Clean up
clean:
	rm -rf obj_dir *.o $(EXEC) trace.vcd

# Auto-generate dependencies
depend:
	@echo "Dependancies are managed by the build rule, no command required"

