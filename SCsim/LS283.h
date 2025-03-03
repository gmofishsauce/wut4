#ifndef LS283_H
#define LS283_H

#include <systemc.h>

SC_MODULE(LS283)
{
    // Control
    sc_in<bool> clock;
    sc_in<bool> RST;

    // Inputs
    sc_in<bool> A1, A2, A3, A4;
    sc_in<bool> B1, B2, B3, B4;
    sc_in<bool> Cin;

    // Outputs
    sc_out<bool> Sum1, Sum2, Sum3, Sum4;
    sc_out<bool> Cout;

    // Process method
    void add();

    SC_CTOR(LS283)
    {
        SC_CTHREAD(add, clock.pos()); // added clock and edge sensitivity
        reset_signal_is(RST, true);
    }
};

#endif
