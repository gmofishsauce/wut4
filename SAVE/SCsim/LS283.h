#ifndef LS283_H
#define LS283_H

#include <systemc.h>

SC_MODULE(LS283)
{
    // Inputs
    sc_in<sc_uint<4>> A;
    sc_in<sc_uint<4>> B;
    sc_in<bool> Cin;

    // Outputs
    sc_out<sc_uint<4>> Sum;
    sc_out<bool> Cout;

    // Process method
    void add(void);

    SC_CTOR(LS283)
    {
        SC_METHOD(add);
        sensitive << A << B << Cin;
    }
};

#endif
