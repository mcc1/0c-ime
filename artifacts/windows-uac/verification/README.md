# Validation scope

The UAC fix branch was rebased onto upstream 0.2.11 before merging into the mixed-input/neighbor preview build. The initial mixed-input failures were obsolete expectations: upstream 0.2.11 deliberately resumes Chinese after alphanumeric prefixes and protects canonical model names. The updated expectations match upstream's own new test cases; production guards were retained.

All file hashes in the signed official 0.2.11 bundle were checked. Both new TSF release DLLs were separately checked for x64/x86 architecture, required COM exports, icon/version resources, ASLR/NX and absence of extra GCC/MinGW runtime DLL dependencies. Brand/version resources are built through windres from the same icon resource generator used by the MSVC build.

The Wine log contains one deliberate FAILED control: the real official 0.2.11 DLL must reject isolated secure activation at the expected COM boundary (E_NOINTERFACE). The test script asserts this exact failure before loading the actual new release DLL, which must pass activation, key pass-through, deactivation and unload. It does not call registration or recreate a secure desktop. This is not a real UAC reproduction.

Helper tests and a full real-official-bundle preparation prove both TSF files are replaced while base UI/data, receipts and user sentinels are retained. Actual Windows elevation, installation/registration, native UAC restoration and rollback remain user-trial items. A Windows sign-out/sign-in is required after this native-shell update. Existing trace/runtime experiments are excluded.
