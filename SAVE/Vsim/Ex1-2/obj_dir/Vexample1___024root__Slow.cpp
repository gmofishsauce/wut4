// Verilated -*- C++ -*-
// DESCRIPTION: Verilator output: Design implementation internals
// See Vexample1.h for the primary calling header

#include "Vexample1__pch.h"
#include "Vexample1__Syms.h"
#include "Vexample1___024root.h"

void Vexample1___024root___ctor_var_reset(Vexample1___024root* vlSelf);

Vexample1___024root::Vexample1___024root(Vexample1__Syms* symsp, const char* v__name)
    : VerilatedModule{v__name}
    , vlSymsp{symsp}
 {
    // Reset structure values
    Vexample1___024root___ctor_var_reset(this);
}

void Vexample1___024root::__Vconfigure(bool first) {
    (void)first;  // Prevent unused variable warning
}

Vexample1___024root::~Vexample1___024root() {
}
