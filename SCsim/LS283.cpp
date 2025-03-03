#include <iostream>
using namespace std;

#include "LS283.h"

void LS283::add()
{
    while (true)
    {
        if (RST.read())
        {
            wait(); // Wait for the next clock edge when reset is active
        }
        else
        {
            wait(); // wait one clock cycle
            // Assemble inputs into integers using bit shifting and OR
            sc_uint<4> a_int = (A4.read() << 3) | (A3.read() << 2) | (A2.read() << 1) | A1.read();
            sc_uint<4> b_int = (B4.read() << 3) | (B3.read() << 2) | (B2.read() << 1) | B1.read();
            sc_uint<1> cin_int = Cin.read();

            // Perform addition
            sc_uint<5> sum_int = a_int + b_int + cin_int;

            // Extract the bits
            Sum1.write(sum_int[0]);
            Sum2.write(sum_int[1]);
            Sum3.write(sum_int[2]);
            Sum4.write(sum_int[3]);
            Cout.write(sum_int[4]);
        }
    }
}
