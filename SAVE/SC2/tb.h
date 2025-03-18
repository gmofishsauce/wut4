#ifndef TB_H
#define TB_H

#include <systemc.h>
#include "mod.h"

SC_MODULE(tb)
{
    sc_signal<sc_uint<4>> sig_A;
    sc_signal<sc_uint<4>> sig_B;
    sc_signal<bool> sig_Cin;
    sc_signal<sc_uint<4>> sig_Sum;
    sc_signal<bool> sig_Cout;

    mod *dut_mod;
    SC_CTOR(tb)
    {
        dut_mod = new mod("dut_mod");
        dut_mod->A(sig_A);
        dut_mod->B(sig_B);
        dut_mod->Cin(sig_Cin);
        dut_mod->Sum(sig_Sum);
        dut_mod->Cout(sig_Cout);
    }
};

SC_MODULE(LS283Tests)
{
public:
    Testbench tb; // The LS283 testbench

    // Constructor
    SC_CTOR(LS283Tests) : tb("tb") {
        SC_METHOD(run);
    }

    // Test functions
    void test1();
};

#endif
