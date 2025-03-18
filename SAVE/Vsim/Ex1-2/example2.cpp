#include <iostream>
using namespace std;

#include "Vexample1.h"
#include "verilated.h"

int main(int argc, char** argv) {
    cout << argv[0] << endl;
    VerilatedContext* contextp = new VerilatedContext;
    contextp->commandArgs(argc, argv);
    Vexample1* top = new Vexample1{contextp};
    while (!contextp->gotFinish()) { top->eval(); }
    delete top;
    delete contextp;
    return 0;
}

