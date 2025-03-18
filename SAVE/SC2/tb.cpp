#include "LS283Tests.h"
#include <iostream>

using namespace sc_core;
using namespace std;

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
    tb.sig_A.write(1);
    wait(17, SC_NS);
    tb.sig_B.write(1);
    tb.sig_Cin.write(false);
}

void LS283Tests::testAdd2()
{
}

void LS283Tests::testAdd3()
{
}

// Run all tests
void LS283Tests::run()
{
    testAdd1();
    testAdd2();
    testAdd3();
}
