# UAC switches 0c to ENG

User evidence: UAC appearance switches the taskbar indicator to ENG; returning requires Win+Space to reselect 0c. Microsoft Zhuyin does not exhibit this in the same user's comparison. The 0c settings process remains open. Engine survival is not yet independently observed.

This is an input-method selection/restoration problem until evidence establishes otherwise, rather than an ordinary 0c Chinese/English mode toggle. Windows per-application input-method selection and 0c mode-memory are separate settings. No mode-memory change is justified by the current evidence.

Code checks:
- The Windows preview uses the unmodified official 0.2.10 native TSF shell; git diff v0.2.10..56a5881 -- platforms/windows is empty. Composition features change the Go engine/settings, not native UAC handling. This narrows the search; it does not rule out every indirect engine interaction.
- register.rs registers six TSF categories and does not include GUID_TFCAT_TIPCAP_SECUREMODE. Microsoft SampleIME includes it: https://github.com/microsoft/Windows-classic-samples/blob/main/Samples/IME/cpp/SampleIME/Register.cpp . The sample is not evidence of Microsoft Zhuyin's exact implementation. Missing secure-mode support is a candidate cause of fallback, not a demonstrated explanation of persistent selection loss.
- ActivateEx records TF_TMAE_SECUREMODE; allowed() blocks forwarding input in that mode. Activate() still warms the candidate helper and probes an existing engine. Adding SECUREMODE alone would advertise behavior this external-engine architecture has not validated on the secure desktop.
- The keyboard open/close notification only toggles internal mode for an observed Ctrl+Space hotkey. It does not choose the Windows ENG keyboard profile.
- Upstream v0.2.11 changes Windows full-width handling; no UAC/profile-restoration fix was identified in its diff/history.

Next evidence: run platforms/windows/test/uac-profile-probe.ps1 as the ordinary desktop user, focus a normal text window with 0c selected, trigger UAC and cancel, return to the same text window before pressing Win+Space. It samples foreground HKL, desktop availability and 0c process IDs, with no input text, titles, key hooks or setting changes. Compare with Microsoft Zhuyin. HKL does not identify a TIP profile uniquely, so corroborate with the taskbar observation and existing activation logs if needed; do not enable tsf-debug.

Acceptance for a future fix: cancel and accept UAC both preserve selection when returning to the original normal text window; manual switching to ENG still works; native/shared mode-memory remain correct; sensitive/secure input does not reach the normal user engine or learning store. Test same-account elevation separately from different-account administrator credentials. Avoid automatic global profile activation as an unverified workaround.

The probe has been source-reviewed only. This Linux environment has no native Windows UAC desktop; neither the probe nor a product fix has been executed on Windows. No product behavior has been changed.
