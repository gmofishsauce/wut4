// Verilated -*- SystemC -*-
// DESCRIPTION: Verilator output: Design implementation internals
// See Vour.h for the primary calling header

#include "Vour__pch.h"
#include "Vour___024root.h"

VL_ATTR_COLD void Vour___024root___eval_static(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_static\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

VL_ATTR_COLD void Vour___024root___eval_initial(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_initial\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    vlSelfRef.__Vtrigprevexpr___TOP____Vcellinp__our__clk__0 
        = vlSelfRef.__Vcellinp__our__clk;
}

VL_ATTR_COLD void Vour___024root___eval_final(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_final\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
}

#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__stl(Vour___024root* vlSelf);
#endif  // VL_DEBUG
VL_ATTR_COLD bool Vour___024root___eval_phase__stl(Vour___024root* vlSelf);

VL_ATTR_COLD void Vour___024root___eval_settle(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_settle\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    IData/*31:0*/ __VstlIterCount;
    CData/*0:0*/ __VstlContinue;
    // Body
    __VstlIterCount = 0U;
    vlSelfRef.__VstlFirstIteration = 1U;
    __VstlContinue = 1U;
    while (__VstlContinue) {
        if (VL_UNLIKELY(((0x64U < __VstlIterCount)))) {
#ifdef VL_DEBUG
            Vour___024root___dump_triggers__stl(vlSelf);
#endif
            VL_FATAL_MT("our.v", 1, "", "Settle region did not converge.");
        }
        __VstlIterCount = ((IData)(1U) + __VstlIterCount);
        __VstlContinue = 0U;
        if (Vour___024root___eval_phase__stl(vlSelf)) {
            __VstlContinue = 1U;
        }
        vlSelfRef.__VstlFirstIteration = 0U;
    }
}

#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__stl(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___dump_triggers__stl\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1U & (~ vlSelfRef.__VstlTriggered.any()))) {
        VL_DBG_MSGF("         No triggers active\n");
    }
    if ((1ULL & vlSelfRef.__VstlTriggered.word(0U))) {
        VL_DBG_MSGF("         'stl' region trigger index 0 is active: Internal 'stl' trigger - first iteration\n");
    }
}
#endif  // VL_DEBUG

void Vour___024root___ico_sequent__TOP__0(Vour___024root* vlSelf);

VL_ATTR_COLD void Vour___024root___eval_stl(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_stl\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1ULL & vlSelfRef.__VstlTriggered.word(0U))) {
        Vour___024root___ico_sequent__TOP__0(vlSelf);
    }
}

VL_ATTR_COLD void Vour___024root___eval_triggers__stl(Vour___024root* vlSelf);

VL_ATTR_COLD bool Vour___024root___eval_phase__stl(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_phase__stl\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Init
    CData/*0:0*/ __VstlExecute;
    // Body
    Vour___024root___eval_triggers__stl(vlSelf);
    __VstlExecute = vlSelfRef.__VstlTriggered.any();
    if (__VstlExecute) {
        Vour___024root___eval_stl(vlSelf);
    }
    return (__VstlExecute);
}

#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__ico(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___dump_triggers__ico\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1U & (~ vlSelfRef.__VicoTriggered.any()))) {
        VL_DBG_MSGF("         No triggers active\n");
    }
    if ((1ULL & vlSelfRef.__VicoTriggered.word(0U))) {
        VL_DBG_MSGF("         'ico' region trigger index 0 is active: Internal 'ico' trigger - first iteration\n");
    }
}
#endif  // VL_DEBUG

#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__act(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___dump_triggers__act\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1U & (~ vlSelfRef.__VactTriggered.any()))) {
        VL_DBG_MSGF("         No triggers active\n");
    }
    if ((1ULL & vlSelfRef.__VactTriggered.word(0U))) {
        VL_DBG_MSGF("         'act' region trigger index 0 is active: @(posedge __Vcellinp__our__clk)\n");
    }
}
#endif  // VL_DEBUG

#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__nba(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___dump_triggers__nba\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    if ((1U & (~ vlSelfRef.__VnbaTriggered.any()))) {
        VL_DBG_MSGF("         No triggers active\n");
    }
    if ((1ULL & vlSelfRef.__VnbaTriggered.word(0U))) {
        VL_DBG_MSGF("         'nba' region trigger index 0 is active: @(posedge __Vcellinp__our__clk)\n");
    }
}
#endif  // VL_DEBUG

VL_ATTR_COLD void Vour___024root___ctor_var_reset(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___ctor_var_reset\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    vlSelf->__Vcellinp__our__clk = VL_RAND_RESET_I(1);
    vlSelf->__Vtrigprevexpr___TOP____Vcellinp__our__clk__0 = VL_RAND_RESET_I(1);
}
