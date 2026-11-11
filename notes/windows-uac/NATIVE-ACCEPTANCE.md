# Native Windows UAC acceptance (pending)

Use a dedicated Windows test account, the normal Windows UAC policy and a package containing rebuilt x64/x86 native shells. Register through the existing installer and sign out/in before testing; an already loaded old DLL is not the patched shell. Do not enable tsf-debug or capture input contents on the UAC desktop.

Compare baseline and candidate in the same ordinary text application with 0c selected. Start with mode-memory=native and normal Chinese output.

- [ ] Trigger same-account UAC, cancel, return to the original text window: 0c remains selected without Win+Space; normal Chinese composition still works.
- [ ] Trigger UAC, accept, return to the original text window: same result. Test typing in the elevated application separately from returning to the original one.
- [ ] Repeat in shared mode. Neither secure activation nor return toggles the remembered Chinese/English state.
- [ ] Deliberately select ENG before and after the transition: it stays selected; the fix must not automatically force 0c.
- [ ] During secure input, verify no candidate helper/engine starts for that secure context and no input or learning request reaches the ordinary user's engine. Existing background processes may remain alive; mere process survival is not an input-forwarding test.
- [ ] Repeat with a 32-bit normal application and with password/private fields.
- [ ] Treat different-account administrator credentials as a distinct case: do not conflate account/profile differences with same-account consent.

Record taskbar profile selection, foreground HKL, native DLL version/path/hash, relevant activation/deactivation events and engine PID continuity. HKL alone does not distinguish every TSF TIP. The included read-only probe can supplement manual taskbar observations but does not itself certify these cases.
