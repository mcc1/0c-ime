//! Exercise the real COM entry points with a manager that refuses all services.
//! Secure pass-through must need neither the manager's focus nor any helper IPC.
use super::*;
use windows::Win32::Foundation::{E_NOTIMPL, HWND};
use windows::Win32::UI::TextServices::{
    ITfThreadMgr_Impl, IEnumTfDocumentMgrs, ITfFunctionProvider,
    IEnumTfFunctionProviders, ITfCompartmentMgr,
};

#[implement(ITfThreadMgr)]
struct NoServices { calls: Cell<u32> }
impl NoServices_Impl {
    fn refuse<T>(&self) -> windows_core::Result<T> {
        self.calls.set(self.calls.get()+1);
        Err(E_NOTIMPL.into())
    }
}
impl ITfThreadMgr_Impl for NoServices_Impl {
    fn Activate(&self) -> windows_core::Result<u32> { self.refuse() }
    fn Deactivate(&self) -> windows_core::Result<()> { self.refuse() }
    fn CreateDocumentMgr(&self) -> windows_core::Result<ITfDocumentMgr> { self.refuse() }
    fn EnumDocumentMgrs(&self) -> windows_core::Result<IEnumTfDocumentMgrs> { self.refuse() }
    fn GetFocus(&self) -> windows_core::Result<ITfDocumentMgr> { self.refuse() }
    fn SetFocus(&self, _: Ref<'_, ITfDocumentMgr>) -> windows_core::Result<()> { self.refuse() }
    fn AssociateFocus(&self, _: HWND, _: Ref<'_, ITfDocumentMgr>) -> windows_core::Result<ITfDocumentMgr> { self.refuse() }
    fn IsThreadFocus(&self) -> windows_core::Result<BOOL> { self.refuse() }
    fn GetFunctionProvider(&self, _: *const GUID) -> windows_core::Result<ITfFunctionProvider> { self.refuse() }
    fn EnumFunctionProviders(&self) -> windows_core::Result<IEnumTfFunctionProviders> { self.refuse() }
    fn GetGlobalCompartment(&self) -> windows_core::Result<ITfCompartmentMgr> { self.refuse() }
}

#[test]
fn secure_activation_events_and_deactivation_never_use_normal_runtime() {
    for extra_flags in [0, TF_TMAE_COMLESS, TF_TMAE_UIELEMENTENABLEDONLY] {
        let manager=ComObject::new(NoServices {calls:Cell::new(0)});
        let tm:ITfThreadMgr=manager.to_interface();
        crate::runtime_activity::take();
        let object=ComObject::new(TextService::new());
        let tip:ITfTextInputProcessorEx=object.to_interface();
        let flags=TF_TMAE_SECUREMODE|extra_flags;
        unsafe {tip.ActivateEx(&tm,7,flags).unwrap();}
        {
            let st=object.state.borrow();
            assert!(st.secure);
            assert_eq!(st.activate_flags,flags);
            assert!(st.callback.is_none() && st.langbar.is_none() && st.openclose.is_none());
        }
        let keys:ITfKeyEventSink=object.to_interface();
        unsafe {
            for vk in [0x41,0x10,0x14,0x20] {
                assert!(!keys.OnTestKeyDown(None,WPARAM(vk),LPARAM(0)).unwrap().as_bool());
                assert!(!keys.OnKeyDown(None,WPARAM(vk),LPARAM(0)).unwrap().as_bool());
                assert!(!keys.OnTestKeyUp(None,WPARAM(vk),LPARAM(0)).unwrap().as_bool());
                assert!(!keys.OnKeyUp(None,WPARAM(vk),LPARAM(0)).unwrap().as_bool());
            }
            keys.OnSetFocus(false).unwrap();
            let focus:ITfThreadMgrEventSink=object.to_interface();
            focus.OnSetFocus(None,None).unwrap();
        }
        object.toggle_mode();
        assert!(!object.set_output_simplified(true));
        object.recheck_simplified();
        unsafe {tip.Deactivate().unwrap();}
        assert!(!object.state.borrow().secure);
        assert_eq!(manager.calls.get(),0);
        assert_eq!(crate::runtime_activity::take(),Vec::<&str>::new());
    }
}

#[test]
fn legacy_activation_after_secure_pass_through_uses_normal_flags() {
    let manager=ComObject::new(NoServices {calls:Cell::new(0)});
    let tm:ITfThreadMgr=manager.to_interface();
    let object=ComObject::new(TextService::new());
    let tip:ITfTextInputProcessorEx=object.to_interface();
    unsafe {
        tip.ActivateEx(&tm,7,TF_TMAE_SECUREMODE).unwrap();
        tip.Deactivate().unwrap();
        // The fake deliberately lacks ITfKeystrokeMgr, so normal activation
        // fails there. It must nevertheless leave secure pass-through mode.
        assert!(tip.Activate(&tm,8).is_err());
    }
    let st=object.state.borrow();
    assert!(!st.secure && !st.ui_less);
    assert_eq!(st.activate_flags,0);
}
