#include <iostream>
#include <systemc.h>
#include "LS283.h"

using namespace sc_core;
using namespace std;

// Simple AND gate module
SC_MODULE(AndGate)
{
  sc_in<bool> a, b;
  sc_out<bool> out;

  void process()
  {
    out.write(a.read() && b.read());
  }

  SC_CTOR(AndGate)
  {
    SC_METHOD(process);
    sensitive << a << b;
  }
};

// Testbench module
SC_MODULE(Testbench)
{
  // Control signals
  sc_signal<bool> clock;
  sc_signal<bool> RST;

  // Combinational data signals
  sc_signal<bool> sig_a, sig_b, sig_out;
  sc_signal<bool> sig_A1, sig_A2, sig_A3, sig_A4;
  sc_signal<bool> sig_B1, sig_B2, sig_B3, sig_B4;
  sc_signal<bool> sig_Cin, sig_Cout;
  sc_signal<bool> sig_Sum1, sig_Sum2, sig_Sum3, sig_Sum4;

  AndGate *dut; // Device Under Test (our AND gate)
  LS283 *dut_LS283;

  void stimulus()
  {
    RST.write(true);
    clock.write(false);
    wait(1, SC_NS);
    clock.write(true);
    wait(1, SC_NS);
    RST.write(false);
    clock.write(false);
    wait(1, SC_NS);
    // Test vector: (0, 0), (0, 1), (1, 0), (1,1)
    sig_a.write(false);
    sig_b.write(false);
    wait(1, SC_NS);
    clock.write(true);
    wait(1, SC_NS);
    cout << "a=" << sig_a.read() << ", b=" << sig_b.read() << ", out=" << sig_out.read() << endl;
    clock.write(false);
    wait(1, SC_NS);

    sig_a.write(false);
    sig_b.write(true);
    clock.write(true);
    wait(1, SC_NS);
    cout << "a=" << sig_a.read() << ", b=" << sig_b.read() << ", out=" << sig_out.read() << endl;
    clock.write(false);
    wait(1, SC_NS);

    sig_a.write(true);
    sig_b.write(false);
    clock.write(true);
    wait(1, SC_NS);
    cout << "a=" << sig_a.read() << ", b=" << sig_b.read() << ", out=" << sig_out.read() << endl;
    clock.write(false);
    wait(1, SC_NS);

    sig_a.write(true);
    sig_b.write(true);
    clock.write(true);
    wait(1, SC_NS);
    cout << "a=" << sig_a.read() << ", b=" << sig_b.read() << ", out=" << sig_out.read() << endl;
    clock.write(false);
    wait(1, SC_NS);

    // LS283 Test
    cout << "LS283 test 1: ";
    sig_A1.write(true);
    sig_A2.write(false);
    sig_A3.write(true);
    sig_A4.write(false);
    sig_B1.write(true);
    sig_B2.write(false);
    sig_B3.write(true);
    sig_B4.write(false);
    sig_Cin.write(true);
    clock.write(true);
    wait(1, SC_NS);
    cout << "@" << sc_time_stamp() << " ";
    cout << "A4,A3,A2,A1 = 0101, B4,B3,B2,B1 = 0101, Cin = 1, Sum4,Sum3,Sum2,Sum1 = " << sig_Sum4.read() << sig_Sum3.read() << sig_Sum2.read() << sig_Sum1.read() << ", Cout = " << sig_Cout.read() << endl;

    clock.write(false);
    wait(1, SC_NS);

    // LS283 Test
    sig_A1.write(false);
    sig_A2.write(false);
    sig_A3.write(false);
    sig_A4.write(false);
    sig_B1.write(false);
    sig_B2.write(false);
    sig_B3.write(false);
    sig_B4.write(false);
    sig_Cin.write(false);
    clock.write(true);
    wait(1, SC_NS);
    cout << "LS283 test 2: ";
    cout << "@" << sc_time_stamp() << " ";
    cout << "A4,A3,A2,A1 = 0000, B4,B3,B2,B1 = 0000, Cin = 0, Sum4,Sum3,Sum2,Sum1 = " << sig_Sum4.read() << sig_Sum3.read() << sig_Sum2.read() << sig_Sum1.read() << ", Cout = " << sig_Cout.read() << endl;
    clock.write(false);
    wait(1, SC_NS);

    sig_A1.write(true);
    sig_A2.write(true);
    sig_A3.write(true);
    sig_A4.write(true);
    sig_B1.write(true);
    sig_B2.write(true);
    sig_B3.write(true);
    sig_B4.write(true);
    sig_Cin.write(true);
    clock.write(true);
    wait(1, SC_NS);
    cout << "LS283 test 3: ";
    cout << "@" << sc_time_stamp() << " ";
    cout << "A4,A3,A2,A1 = 1111, B4,B3,B2,B1 = 1111, Cin = 1, Sum4,Sum3,Sum2,Sum1 = " << sig_Sum4.read() << sig_Sum3.read() << sig_Sum2.read() << sig_Sum1.read() << ", Cout = " << sig_Cout.read() << endl;
    clock.write(false);
    wait(1, SC_NS);

    sc_stop(); // Stop the simulation after the test vectors
  }

  SC_CTOR(Testbench)
  {
    dut = new AndGate("dut");
    dut->a(sig_a);
    dut->b(sig_b);
    dut->out(sig_out);

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

    SC_THREAD(stimulus);
  }
};

int sc_main(int argc, char *argv[])
{
  Testbench tb("tb");
  sc_start(30, SC_NS);
  return 0;
}
