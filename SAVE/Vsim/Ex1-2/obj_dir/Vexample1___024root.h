// Verilated -*- C++ -*-
// DESCRIPTION: Verilator output: Design internal header
// See Vexample1.h for the primary calling header

#ifndef VERILATED_VEXAMPLE1___024ROOT_H_
#define VERILATED_VEXAMPLE1___024ROOT_H_  // guard

#include "verilated.h"


class Vexample1__Syms;

class alignas(VL_CACHE_LINE_BYTES) Vexample1___024root final : public VerilatedModule {
  public:

    // DESIGN SPECIFIC STATE
    CData/*0:0*/ __VactContinue;
    IData/*31:0*/ __VactIterCount;
    VlTriggerVec<0> __VactTriggered;
    VlTriggerVec<0> __VnbaTriggered;

    // INTERNAL VARIABLES
    Vexample1__Syms* const vlSymsp;

    // CONSTRUCTORS
    Vexample1___024root(Vexample1__Syms* symsp, const char* v__name);
    ~Vexample1___024root();
    VL_UNCOPYABLE(Vexample1___024root);

    // INTERNAL METHODS
    void __Vconfigure(bool first);
};


#endif  // guard
