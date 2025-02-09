package testingwrap

import "runtime"

type StackCache interface {
	GetCaller() runtime.Frame
	GetStackFrames() []runtime.Frame
}

func GetCaller(st StackCache) runtime.Frame {
	return st.GetCaller()
}

func GetStackFrames(st StackCache) []runtime.Frame {
	return st.GetStackFrames()
}

func WrapCall(st StackCache, fn func(st StackCache)) {
	fn(st)
}
