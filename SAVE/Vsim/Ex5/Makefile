VERILATOR = verilator
VERILATOR_FLAGS = -Wall --cc --trace

all: sim

sim: and_gate.v
	$(VERILATOR) $(VERILATOR_FLAGS) and_gate.v --exe test_and_gate.cpp
	make -C obj_dir -f Vand_gate.mk Vand_gate

run: sim
	obj_dir/Vand_gate

clean:
	rm -rf obj_dir trace.vcd

.PHONY: all sim run clean