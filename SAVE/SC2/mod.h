#ifndef MOD_H
#define MOD_H

#include <systemc.h>

SC_MODULE(mod)
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

    SC_CTOR(mod)
    {
        SC_METHOD(add);
        sensitive << A << B << Cin;
    }
};

#endif
