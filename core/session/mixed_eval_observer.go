package session

import "github.com/0c-biggo/0c-ime/core/zhuyin"

// Nil in ordinary builds. The offline adapter observes insertion provenance;
// neither callback can alter the insertion result or parser branch selection.
var mixedEvalObserveInsert func(*Session, zhuyin.Syllable, string) func()
