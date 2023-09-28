package gad

func (vm *VM) Write(b []byte) (int, error) {
	if l := len(vm.writers); l > 0 {
		return vm.writers[l-1].Write(b)
	}
	return vm.StdOut.Write(b)
}

func (vm *VM) Read(b []byte) (int, error) {
	return vm.StdIn.Read(b)
}
