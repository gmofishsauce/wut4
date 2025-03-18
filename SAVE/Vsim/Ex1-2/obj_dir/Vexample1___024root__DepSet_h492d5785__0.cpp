// Verilated -*- C++ -*-
// DESCRIPTION: Verilator output: Design implementation internals
// See Vexample1.h for the primary calling header

#include "Vexample1__pch.h"
#include "Vexample1__Syms.h"
#include "Vexample1___024root.h"

#ifdef VL_DEBUG
VL_ATTR_COLD void Vexample1___024root___dump_triggers__act(Vexample1___024root* vlSelf);
#endif  // VL_DEBUG

void Vexample1___024root___eval_triggers__act(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_triggers__act\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
#ifdef VL_DEBUG
    if (VL_UNLIKELY(vlSymsp->_vm_contextp__->debug())) {
        Vexample1___024root___dump_triggers__act(vlSelf);
    }
#endif
}
