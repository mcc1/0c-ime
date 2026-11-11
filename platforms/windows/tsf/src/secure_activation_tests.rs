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

/// Run explicitly against the exact release DLL that will enter the package.
/// This uses its exported COM factory, not the test executable's TextService.
#[test]
#[ignore = "requires ZUIN_TEST_SECURE_DLL pointing to a built release DLL"]
fn release_dll_secure_activation() {
    use windows::Win32::System::Com::IClassFactory;
    use windows::Win32::System::LibraryLoader::{GetProcAddress, LoadLibraryW};
    use windows_core::{HRESULT, PCSTR, PCWSTR};
    let path = std::env::var("ZUIN_TEST_SECURE_DLL").expect("ZUIN_TEST_SECURE_DLL");
    let wide = zuin_win_common::wide(&path);
    // The isolated probe process owns this module until it exits.
    let module = unsafe { LoadLibraryW(PCWSTR(wide.as_ptr())).unwrap() };
    type GetClass = unsafe extern "system" fn(*const GUID, *const GUID, *mut *mut std::ffi::c_void) -> HRESULT;
    type CanUnload = unsafe extern "system" fn() -> HRESULT;
    let get: GetClass = unsafe { std::mem::transmute(GetProcAddress(module, PCSTR(b"DllGetClassObject\0".as_ptr())).unwrap()) };
    let can_unload: CanUnload = unsafe { std::mem::transmute(GetProcAddress(module, PCSTR(b"DllCanUnloadNow\0".as_ptr())).unwrap()) };
    let mut raw = std::ptr::null_mut();
    unsafe { get(&crate::guids::CLSID_ZUIN, &IClassFactory::IID, &mut raw).ok().unwrap(); }
    let factory = unsafe { IClassFactory::from_raw(raw) };
    let tip: ITfTextInputProcessorEx = unsafe { factory.CreateInstance(None).unwrap() };
    let manager = ComObject::new(NoServices { calls: Cell::new(0) });
    let tm: ITfThreadMgr = manager.to_interface();
    unsafe { tip.ActivateEx(&tm, 7, TF_TMAE_SECUREMODE).expect("release DLL must support isolated secure activation"); }
    let keys: ITfKeyEventSink = tip.cast().unwrap();
    unsafe {
        assert!(!keys.OnTestKeyDown(None, WPARAM(0x41), LPARAM(0)).unwrap().as_bool());
        assert!(!keys.OnKeyDown(None, WPARAM(0x41), LPARAM(0)).unwrap().as_bool());
        tip.Deactivate().unwrap();
    }
    assert_eq!(manager.calls.get(), 0);
    drop(keys);
    drop(tip);
    drop(factory);
    assert_eq!(unsafe { can_unload() }, windows::Win32::Foundation::S_OK);
}
