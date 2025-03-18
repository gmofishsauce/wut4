// Verilated -*- C++ -*-
// DESCRIPTION: Verilator output: Design implementation internals
// See Vexample1.h for the primary calling header

#include "Vexample1__pch.h"
#include "Vexample1___024root.h"

VL_ATTR_COLD void Vexample1___024root___eval_static(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_static\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

VL_ATTR_COLD void Vexample1___024root___eval_initial__TOP(Vexample1___024root* vlSelf);

VL_ATTR_COLD void Vexample1___024root___eval_initial(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_initial\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    Vexample1___024root___eval_initial__TOP(vlSelf);
}

VL_ATTR_COLD void Vexample1___024root___eval_initial__TOP(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_initial__TOP\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    VL_WRITEF_NX("Hello World\n",0);
    VL_FINISH_MT("example1.v", 2, "");
}

VL_ATTR_COLD void Vexample1___024root___eval_final(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_final\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

VL_ATTR_COLD void Vexample1___024root___eval_settle(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_settle\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

#ifdef VL_DEBUG
VL_ATTR_COLD void Vexample1___024root___dump_triggers__act(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___dump_triggers__act\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1U & (~ vlSelfRef.__VactTriggered.any()))) {
        VL_DBG_MSGF("         No triggers active\n");
    }
}
#endif  // VL_DEBUG

#ifdef VL_DEBUG
VL_ATTR_COLD void Vexample1___024root___dump_triggers__nba(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___dump_triggers__nba\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1U & (~ vlSelfRef.__VnbaTriggered.any()))) {
        VL_DBG_MSGF("         No triggers active\n");
    }
}
#endif  // VL_DEBUG

VL_ATTR_COLD void Vexample1___024root___ctor_var_reset(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___ctor_var_reset\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}
