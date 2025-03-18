#ifndef LS283TESTS_H
#define LS283TESTS_H

#include <systemc.h>
#include "LS283.h"

SC_MODULE(Testbench)
{
    sc_signal<sc_uint<4>> sig_A;
    sc_signal<sc_uint<4>> sig_B;
    sc_signal<bool> sig_Cin;
    sc_signal<sc_uint<4>> sig_Sum;
    sc_signal<bool> sig_Cout;

    LS283 *dut_LS283;
    SC_CTOR(Testbench)
    {
        dut_LS283 = new LS283("dut_LS283");
        dut_LS283->A(sig_A);
        dut_LS283->B(sig_B);
        dut_LS283->Cin(sig_Cin);
        dut_LS283->Sum(sig_Sum);
        dut_LS283->Cout(sig_Cout);
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

    // Assertion macros
    void ASSERT_TRUE(bool condition, const char *message);
    void ASSERT_EQ(bool expected, bool actual, const char *message);
    void ASSERT_EQ_INT(int expected, int actual, const char *message);
    // Test functions
    void testAdd1();
    void testAdd2();
    void testAdd3();

    // Run all tests
    void run();
};

#endif
