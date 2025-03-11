#ifndef LS283TESTS_H
#define LS283TESTS_H

#include <systemc.h>
#include "LS283.h"
// Testbench module
SC_MODULE(Testbench)
{
  // Control signals
  sc_signal<bool> clock;
  sc_signal<bool> RST;

  // Combinational data signals
  sc_signal<bool> sig_A1, sig_A2, sig_A3, sig_A4;
  sc_signal<bool> sig_B1, sig_B2, sig_B3, sig_B4;
  sc_signal<bool> sig_Cin, sig_Cout;
  sc_signal<bool> sig_Sum1, sig_Sum2, sig_Sum3, sig_Sum4;

  LS283 *dut_LS283;
  SC_CTOR(Testbench)
  {

    dut_LS283 = new LS283("dut_LS283");
    dut_LS283->A1(sig_A1);
    dut_LS283->A2(sig_A2);
    dut_LS283->A3(sig_A3);
    dut_LS283->A4(sig_A4);
    dut_LS283->B1(sig_B1);
    dut_LS283->B2(sig_B2);
    dut_LS283->B3(sig_B3);
    dut_LS283->B4(sig_B4);
    dut_LS283->Cin(sig_Cin);
    dut_LS283->Sum1(sig_Sum1);
    dut_LS283->Sum2(sig_Sum2);
    dut_LS283->Sum3(sig_Sum3);
    dut_LS283->Sum4(sig_Sum4);
    dut_LS283->Cout(sig_Cout);
    dut_LS283->clock(clock);
    dut_LS283->RST(RST);
  }
};

class LS283Tests
{
public:
  Testbench tb; // The LS283 testbench

  // Constructor
  LS283Tests();

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
