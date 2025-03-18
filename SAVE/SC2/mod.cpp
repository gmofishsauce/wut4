#include <iostream>
using namespace std;

#include "LS283.h"

void LS283::add()
{
    cout << "@" << sc_time_stamp() << " A = " << A.read() << "B = " << B.read()
         << "Cin = " << Cin.read() << endl;

    /*
    while (true)
    {
        sc_uint<1> cin_int = Cin.read();
        sc_uint<5> sum_int = A.read() + B.read() + cin_int;
        cout << "LS238::add(): waiting @" << sc_time_stamp() << endl;
        wait(5, SC_NS);
        cout << "LS238::add(): writing @" << sc_time_stamp() << endl;
        // Extract the bits
        Sum.write(sum_int);
        Cout.write(sum_int[4]);
    }
    */
}
