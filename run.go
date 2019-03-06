package cas9

import (
	"syscall/js"
)

func StartApp(targetSelector string, mainComp ComponentView) {
	wait := make(chan struct{}, 0)

	js.Global().Set("sayHi", js.NewCallback(func(params []js.Value) {
		println("WASM Go Initialized")
	}))

	AttachTo(targetSelector, mainComp)

	<-wait
}
