#include <iostream>
#include <systemc.h>
#include "LS283.h"
#include "LS283Tests.h"

using namespace sc_core;
using namespace std;

int sc_main(int argc, char *argv[])
{
  LS283Tests tests;
  cout << "calling tests.run()" << endl;
  tests.run();
  cout << "calling sc_start()" << endl;
  sc_start(30, SC_NS);
  cout << "done" << endl;
  return 0;
}
