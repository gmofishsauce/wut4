// Verilated -*- SystemC -*-
// DESCRIPTION: Verilator output: Design implementation internals
// See Vour.h for the primary calling header

#include "Vour__pch.h"
#include "Vour__Syms.h"
#include "Vour___024root.h"

#ifdef VL_DEBUG
VL_ATTR_COLD void Vour___024root___dump_triggers__stl(Vour___024root* vlSelf);
#endif  // VL_DEBUG

VL_ATTR_COLD void Vour___024root___eval_triggers__stl(Vour___024root* vlSelf) {
    VL_DEBUG_IF(VL_DBG_MSGF("+    Vour___024root___eval_triggers__stl\n"); );
    Vour__Syms* const __restrict vlSymsp VL_ATTR_UNUSED = vlSelf->vlSymsp;
    auto& vlSelfRef = std::ref(*vlSelf).get();
    // Body
    vlSelfRef.__VstlTriggered.set(0U, (IData)(vlSelfRef.__VstlFirstIteration));
#ifdef VL_DEBUG
    if (VL_UNLIKELY(vlSymsp->_vm_contextp__->debug())) {
        Vour___024root___dump_triggers__stl(vlSelf);
    }
#endif
}
