// Verilated -*- C++ -*-
// DESCRIPTION: Verilator output: Symbol table internal header
//
// Internal details; most calling programs do not need this header,
// unless using verilator public meta comments.

#ifndef VERILATED_VEXAMPLE1__SYMS_H_
#define VERILATED_VEXAMPLE1__SYMS_H_  // guard

#include "verilated.h"

// INCLUDE MODEL CLASS

#include "Vexample1.h"

// INCLUDE MODULE CLASSES
#include "Vexample1___024root.h"

// SYMS CLASS (contains all model state)
class alignas(VL_CACHE_LINE_BYTES)Vexample1__Syms final : public VerilatedSyms {
  public:
    // INTERNAL STATE
    Vexample1* const __Vm_modelp;
    VlDeleter __Vm_deleter;
    bool __Vm_didInit = false;

    // MODULE INSTANCE STATE
    Vexample1___024root            TOP;

    // CONSTRUCTORS
    Vexample1__Syms(VerilatedContext* contextp, const char* namep, Vexample1* modelp);
    ~Vexample1__Syms();

    // METHODS
    const char* name() { return TOP.name(); }
};

#endif  // guard
