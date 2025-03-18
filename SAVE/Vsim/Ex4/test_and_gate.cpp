#include <iostream>
#include <verilated.h>
#include <verilated_vcd_c.h>
#include "Vand_gate.h" // Include the Verilated header for our AND gate

int main(int argc, char **argv, char **env)
{
    // Initialize Verilators variables
    Verilated::commandArgs(argc, argv);

    // Create instance of module
    Vand_gate *top = new Vand_gate;

    // Tracefile
    Verilated::traceEverOn(true);
    VerilatedVcdC *tfp = new VerilatedVcdC;
    top->trace(tfp, 99);
    tfp->open("trace.vcd");

    // Test Cases
    for (int a = 0; a < 2; ++a)
    {
        for (int b = 0; b < 2; ++b)
        {
            top->a = a;
            top->b = b;
            top->eval();

            tfp->dump(Verilated::time());
            std::cout << "a = " << a << ", b = " << b << ", y = " << top->y << std::endl;
            Verilated::timeInc(1);
        }
    }
    tfp->dump(Verilated::time());

    // Final model cleanup
    top->final();

    // Close trace file
    tfp->close();

    // Destroy model
    delete top;
    delete tfp;
    return 0;
}
