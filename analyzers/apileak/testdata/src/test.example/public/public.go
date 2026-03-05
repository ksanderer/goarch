package public

import "test.example/internal/executor"

func Good() string { return "ok" }

func Bad() *executor.Runner { return nil } // want `\[apileak\] public API must not expose internal type test.example/internal/executor.Runner`

func BadArg(r *executor.Runner) {} // want `\[apileak\] public API must not expose internal type test.example/internal/executor.Runner`

func private() *executor.Runner { return nil } // OK — unexported
