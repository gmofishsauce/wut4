// Verilated -*- C++ -*-
// DESCRIPTION: Verilator output: Design implementation internals
// See Vexample1.h for the primary calling header

#include "Vexample1__pch.h"
#include "Vexample1___024root.h"

void Vexample1___024root___eval_act(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_act\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

void Vexample1___024root___eval_nba(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_nba\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

void Vexample1___024root___eval_triggers__act(Vexample1___024root* vlSelf);

bool Vexample1___024root___eval_phase__act(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_phase__act\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    VlTriggerVec<0> __VpreTriggered;
    CData/*0:0*/ __VactExecute;
    // Body
    Vexample1___024root___eval_triggers__act(vlSelf);
    __VactExecute = vlSelfRef.__VactTriggered.any();
    if (__VactExecute) {
        __VpreTriggered.andNot(vlSelfRef.__VactTriggered, vlSelfRef.__VnbaTriggered);
        vlSelfRef.__VnbaTriggered.thisOr(vlSelfRef.__VactTriggered);
        Vexample1___024root___eval_act(vlSelf);
    }
    return (__VactExecute);
}

bool Vexample1___024root___eval_phase__nba(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_phase__nba\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    CData/*0:0*/ __VnbaExecute;
    // Body
    __VnbaExecute = vlSelfRef.__VnbaTriggered.any();
    if (__VnbaExecute) {
        Vexample1___024root___eval_nba(vlSelf);
        vlSelfRef.__VnbaTriggered.clear();
    }
    return (__VnbaExecute);
}

#ifdef VL_DEBUG
VL_ATTR_COLD void Vexample1___024root___dump_triggers__nba(Vexample1___024root* vlSelf);
#endif  // VL_DEBUG
#ifdef VL_DEBUG
VL_ATTR_COLD void Vexample1___024root___dump_triggers__act(Vexample1___024root* vlSelf);
#endif  // VL_DEBUG

void Vexample1___024root___eval(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    IData/*31:0*/ __VnbaIterCount;
    CData/*0:0*/ __VnbaContinue;
    // Body
    __VnbaIterCount = 0U;
    __VnbaContinue = 1U;
    while (__VnbaContinue) {
        if (VL_UNLIKELY(((0x64U < __VnbaIterCount)))) {
#ifdef VL_DEBUG
            Vexample1___024root___dump_triggers__nba(vlSelf);
#endif
            VL_FATAL_MT("example1.v", 1, "", "NBA region did not converge.");
        }
        __VnbaIterCount = ((IData)(1U) + __VnbaIterCount);
        __VnbaContinue = 0U;
        vlSelfRef.__VactIterCount = 0U;
        vlSelfRef.__VactContinue = 1U;
        while (vlSelfRef.__VactContinue) {
            if (VL_UNLIKELY(((0x64U < vlSelfRef.__VactIterCount)))) {
#ifdef VL_DEBUG
                Vexample1___024root___dump_triggers__act(vlSelf);
#endif
                VL_FATAL_MT("example1.v", 1, "", "Active region did not converge.");
            }
            vlSelfRef.__VactIterCount = ((IData)(1U) 
                                         + vlSelfRef.__VactIterCount);
            vlSelfRef.__VactContinue = 0U;
            if (Vexample1___024root___eval_phase__act(vlSelf)) {
                vlSelfRef.__VactContinue = 1U;
            }
        }
        if (Vexample1___024root___eval_phase__nba(vlSelf)) {
            __VnbaContinue = 1U;
        }
    }
}

#ifdef VL_DEBUG
void Vexample1___024root___eval_debug_assertions(Vexample1___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vexample1___024root___eval_debug_assertions\n"); );
    Vexample1__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}
#endif  // VL_DEBUG
