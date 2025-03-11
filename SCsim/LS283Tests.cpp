#include "LS283Tests.h"
#include <iostream>

using namespace sc_core;
using namespace std;

// Constructor
LS283Tests::LS283Tests() : tb("tb")
{
}

// Assertion macros
void LS283Tests::ASSERT_TRUE(bool condition, const char *message)
{
    if (!condition)
    {
        cout << "FAIL: " << message << endl;
    }
    else
    {
        // cout << "PASS: " << message << endl;
    }
}

void LS283Tests::ASSERT_EQ(bool expected, bool actual, const char *message)
{
    if (expected != actual)
    {
        cout << "FAIL: " << message << endl;
    }
    else
    {
        // cout << "PASS: " << message << endl;
    }
}
void LS283Tests::ASSERT_EQ_INT(int expected, int actual, const char *message)
{
    if (expected != actual)
    {
        cout << "FAIL: " << message << endl;
    }
    else
    {
        // cout << "PASS: " << message << endl;
    }
}

// Test functions
void LS283Tests::testAdd1()
{
    cout << "LS283 test 1: ";
    tb.sig_A1.write(true);
    tb.sig_A2.write(false);
    tb.sig_A3.write(true);
    tb.sig_A4.write(false);
    tb.sig_B1.write(true);
    tb.sig_B2.write(false);
    tb.sig_B3.write(true);
    tb.sig_B4.write(false);
    tb.sig_Cin.write(true);
    tb.clock.write(true);
    wait(1, SC_NS);
    cout << "@" << sc_time_stamp() << " ";
    cout << "A4,A3,A2,A1 = 0101, B4,B3,B2,B1 = 0101, Cin = 1, Sum4,Sum3,Sum2,Sum1 = " << tb.sig_Sum4.read() << tb.sig_Sum3.read() << tb.sig_Sum2.read() << tb.sig_Sum1.read() << ", Cout = " << tb.sig_Cout.read() << endl;
    ASSERT_EQ(true, tb.sig_Sum4.read(), "Sum4");
    ASSERT_EQ(false, tb.sig_Sum3.read(), "Sum3");
    ASSERT_EQ(true, tb.sig_Sum2.read(), "Sum2");
    ASSERT_EQ(true, tb.sig_Sum1.read(), "Sum1");
    ASSERT_EQ(true, tb.sig_Cout.read(), "Cout");
    tb.clock.write(false);
    wait(1, SC_NS);
}

void LS283Tests::testAdd2()
{
    cout << "LS283 test 2: ";
    tb.sig_A1.write(false);
    tb.sig_A2.write(false);
    tb.sig_A3.write(false);
    tb.sig_A4.write(false);
    tb.sig_B1.write(false);
    tb.sig_B2.write(false);
    tb.sig_B3.write(false);
    tb.sig_B4.write(false);
    tb.sig_Cin.write(false);
    tb.clock.write(true);
    wait(1, SC_NS);
    cout << "@" << sc_time_stamp() << " ";
    cout << "A4,A3,A2,A1 = 0000, B4,B3,B2,B1 = 0000, Cin = 0, Sum4,Sum3,Sum2,Sum1 = " << tb.sig_Sum4.read() << tb.sig_Sum3.read() << tb.sig_Sum2.read() << tb.sig_Sum1.read() << ", Cout = " << tb.sig_Cout.read() << endl;
    ASSERT_EQ(false, tb.sig_Sum4.read(), "Sum4");
    ASSERT_EQ(false, tb.sig_Sum3.read(), "Sum3");
    ASSERT_EQ(false, tb.sig_Sum2.read(), "Sum2");
    ASSERT_EQ(false, tb.sig_Sum1.read(), "Sum1");
    ASSERT_EQ(false, tb.sig_Cout.read(), "Cout");
    tb.clock.write(false);
    wait(1, SC_NS);
}

void LS283Tests::testAdd3()
{
    cout << "LS283 test 3: ";
    tb.sig_A1.write(true);
    tb.sig_A2.write(true);
    tb.sig_A3.write(true);
    tb.sig_A4.write(true);
    tb.sig_B1.write(true);
    tb.sig_B2.write(true);
    tb.sig_B3.write(true);
    tb.sig_B4.write(true);
    tb.sig_Cin.write(true);
    tb.clock.write(true);
    wait(1, SC_NS);
    cout << "@" << sc_time_stamp() << " ";
    cout << "A4,A3,A2,A1 = 1111, B4,B3,B2,B1 = 1111, Cin = 1, Sum4,Sum3,Sum2,Sum1 = " << tb.sig_Sum4.read() << tb.sig_Sum3.read() << tb.sig_Sum2.read() << tb.sig_Sum1.read() << ", Cout = " << tb.sig_Cout.read() << endl;
    ASSERT_EQ(true, tb.sig_Sum4.read(), "Sum4");
    ASSERT_EQ(true, tb.sig_Sum3.read(), "Sum3");
    ASSERT_EQ(true, tb.sig_Sum2.read(), "Sum2");
    ASSERT_EQ(true, tb.sig_Sum1.read(), "Sum1");
    ASSERT_EQ(true, tb.sig_Cout.read(), "Cout");
    tb.clock.write(false);
    wait(1, SC_NS);
}

// Run all tests
void LS283Tests::run()
{
    // reset the system
    tb.RST.write(true);
    tb.clock.write(false);
    wait(1, SC_NS);
    tb.clock.write(true);
    wait(1, SC_NS);
    tb.RST.write(false);
    tb.clock.write(false);
    wait(1, SC_NS);
    testAdd1();
    testAdd2();
    testAdd3();
}
