// Verilated -*- SystemC -*-
// DESCRIPTION: Verilator output: Design implementation internals
// See Vour.h for the primary calling header

#include "Vour__pch.h"
#include "Vour___024root.h"

void Vour___024root___ico_sequent__TOP__0(Vour___024root* vlSelf);

void Vour___024root___eval_ico(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_ico\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1ULL & vlSelfRef.__VicoTriggered.word(0U))) {
        Vour___024root___ico_sequent__TOP__0(vlSelf);
    }
}

VL_INLINE_OPT void Vour___024root___ico_sequent__TOP__0(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___ico_sequent__TOP__0\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    VL_ASSIGN_ISI(1,vlSelfRef.__Vcellinp__our__clk, vlSelfRef.clk);
}

void Vour___024root___eval_triggers__ico(Vour___024root* vlSelf);

bool Vour___024root___eval_phase__ico(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_phase__ico\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    CData/*0:0*/ __VicoExecute;
    // Body
    Vour___024root___eval_triggers__ico(vlSelf);
    __VicoExecute = vlSelfRef.__VicoTriggered.any();
    if (__VicoExecute) {
        Vour___024root___eval_ico(vlSelf);
    }
    return (__VicoExecute);
}

void Vour___024root___eval_act(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_act\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

void Vour___024root___nba_sequent__TOP__0(Vour___024root* vlSelf);

void Vour___024root___eval_nba(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_nba\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1ULL & vlSelfRef.__VnbaTriggered.word(0U))) {
        Vour___024root___nba_sequent__TOP__0(vlSelf);
    }
}

VL_INLINE_OPT void Vour___024root___nba_sequent__TOP__0(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___nba_sequent__TOP__0\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    VL_WRITEF_NX("Hello World\n",0);
    VL_FINISH_MT("our.v", 4, "");
}

void Vour___024root___eval_triggers__act(Vour___024root* vlSelf);

bool Vour___024root___eval_phase__act(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_phase__act\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    VlTriggerVec<1> __VpreTriggered;
    CData/*0:0*/ __VactExecute;
    // Body
    Vour___024root___eval_triggers__act(vlSelf);
    __VactExecute = vlSelfRef.__VactTriggered.any();
    if (__VactExecute) {
        __VpreTriggered.andNot(vlSelfRef.__VactTriggered, vlSelfRef.__VnbaTriggered);
        vlSelfRef.__VnbaTriggered.thisOr(vlSelfRef.__VactTriggered);
        Vour___024root___eval_act(vlSelf);
    }
    return (__VactExecute);
}

bool Vour___024root___eval_phase__nba(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_phase__nba\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    CData/*0:0*/ __VnbaExecute;
    // Body
    __VnbaExecute = vlSelfRef.__VnbaTriggered.any();
    if (__VnbaExecute) {
        Vour___024root___eval_nba(vlSelf);
        vlSelfRef.__VnbaTriggered.clear();
    }
    return (__VnbaExecute);
}

#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__ico(Vour___024root* vlSelf);
#endif  // VL_DEBUG
#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__nba(Vour___024root* vlSelf);
#endif  // VL_DEBUG
#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__act(Vour___024root* vlSelf);
#endif  // VL_DEBUG

void Vour___024root___eval(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    IData/*31:0*/ __VicoIterCount;
    CData/*0:0*/ __VicoContinue;
    IData/*31:0*/ __VnbaIterCount;
    CData/*0:0*/ __VnbaContinue;
    // Body
    __VicoIterCount = 0U;
    vlSelfRef.__VicoFirstIteration = 1U;
    __VicoContinue = 1U;
    while (__VicoContinue) {
        if (VL_UNLIKELY(((0x64U < __VicoIterCount)))) {
#ifdef VL_DEBUG
            Vour___024root___dump_triggers__ico(vlSelf);
#endif
            VL_FATAL_MT("our.v", 1, "", "Input combinational region did not converge.");
        }
        __VicoIterCount = ((IData)(1U) + __VicoIterCount);
        __VicoContinue = 0U;
        if (Vour___024root___eval_phase__ico(vlSelf)) {
            __VicoContinue = 1U;
        }
        vlSelfRef.__VicoFirstIteration = 0U;
    }
    __VnbaIterCount = 0U;
    __VnbaContinue = 1U;
    while (__VnbaContinue) {
        if (VL_UNLIKELY(((0x64U < __VnbaIterCount)))) {
#ifdef VL_DEBUG
            Vour___024root___dump_triggers__nba(vlSelf);
#endif
            VL_FATAL_MT("our.v", 1, "", "NBA region did not converge.");
        }
        __VnbaIterCount = ((IData)(1U) + __VnbaIterCount);
        __VnbaContinue = 0U;
        vlSelfRef.__VactIterCount = 0U;
        vlSelfRef.__VactContinue = 1U;
        while (vlSelfRef.__VactContinue) {
            if (VL_UNLIKELY(((0x64U < vlSelfRef.__VactIterCount)))) {
#ifdef VL_DEBUG
                Vour___024root___dump_triggers__act(vlSelf);
#endif
                VL_FATAL_MT("our.v", 1, "", "Active region did not converge.");
            }
            vlSelfRef.__VactIterCount = ((IData)(1U) 
                                         + vlSelfRef.__VactIterCount);
            vlSelfRef.__VactContinue = 0U;
            if (Vour___024root___eval_phase__act(vlSelf)) {
                vlSelfRef.__VactContinue = 1U;
            }
        }
        if (Vour___024root___eval_phase__nba(vlSelf)) {
            __VnbaContinue = 1U;
        }
    }
}

#ifdef VL_DEBUG
void Vour___024root___eval_debug_assertions(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_debug_assertions\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}
#endif  // VL_DEBUG
