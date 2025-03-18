#include <iostream>
#include <systemc.h>
#include "LS283.h"
#include "LS283Tests.h"

using namespace sc_core;
using namespace std;

int sc_main(int argc, char *argv[])
{
    LS283Tests* tests = new LS283Tests("LS283Tests");

    sc_clock clk("Clock", 100, SC_NS);

    //tests->run();
    sc_start(999, SC_NS);
    return 0;
}
