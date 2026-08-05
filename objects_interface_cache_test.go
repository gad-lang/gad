package gad

import "testing"

// TestInterfaceSatCache checks the interface-satisfaction memoization on the root
// VM: which values are cacheable, that a pre-seeded entry short-circuits the
// check, that a dict is never cached, that sub-VMs share the root's cache, and
// that a host-provided cache can be injected. Internal test: it inspects the
// unexported cache key.
func TestInterfaceSatCache(t *testing.T) {
	builtins := NewBuiltins()
	vm := NewVM(builtins.Build(), &Bytecode{Main: &CompiledFunction{}})

	iface := &Interface{IName: "Named", Fields: []*InterfaceField{{Name: "Name"}}}
	iface.Fields[0].Iface = iface

	// ifaceCacheableType: reflect values are cacheable, dicts are not.
	rv, err := NewReflectValue(struct{ Name string }{Name: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ifaceCacheableType(rv); !ok {
		t.Fatal("a reflect value should be cacheable")
	}
	if _, ok := ifaceCacheableType(Dict{"Name": Str("a")}); ok {
		t.Fatal("a dict must not be cacheable")
	}

	// A dict check does not populate the cache.
	if ok, _ := iface.CanAssignVM(vm, Dict{"Name": Str("a")}); !ok {
		t.Fatal("dict with Name should satisfy")
	}
	if n := vm.interfaceSatCache().Len(); n != 0 {
		t.Fatalf("dict must not be cached, cache=%d", n)
	}

	// A pre-seeded cache entry short-circuits the check (no satisfaction run):
	// seed FALSE and confirm the reflect value is reported unsatisfied from cache.
	key := ifaceSatKey{iface: iface, typ: rv.Type()}
	vm.ifaceSatPut(key, false)
	if ok, err := iface.CanAssignVM(vm, rv); err != nil || ok {
		t.Fatalf("cached FALSE should short-circuit: ok=%v err=%v", ok, err)
	}

	// A sub-VM shares the root's cache (reads through pool.root).
	child := vm.pool.acquire(&CompiledFunction{}, false)
	if v, hit := child.ifaceSatGet(key); !hit || v {
		t.Fatalf("sub-VM should see the root's cache entry: hit=%v v=%v", hit, v)
	}
	// A write from the sub-VM lands on the root.
	key2 := ifaceSatKey{iface: iface, typ: TInt}
	child.ifaceSatPut(key2, true)
	if v, hit := vm.ifaceSatGet(key2); !hit || !v {
		t.Fatal("a sub-VM write should be visible on the root")
	}
}

// TestInterfaceSatCacheInjected checks that a cache built outside the VM and
// injected via SetInterfaceSatCache is used by the root VM.
func TestInterfaceSatCacheInjected(t *testing.T) {
	builtins := NewBuiltins()
	vm := NewVM(builtins.Build(), &Bytecode{Main: &CompiledFunction{}})

	cache := NewInterfaceSatCache()
	iface := &Interface{IName: "Named"}
	cache.put(ifaceSatKey{iface: iface, typ: TInt}, true)

	vm.SetInterfaceSatCache(cache)
	if vm.interfaceSatCache() != cache {
		t.Fatal("injected cache should be the root's cache")
	}
	if v, hit := vm.ifaceSatGet(ifaceSatKey{iface: iface, typ: TInt}); !hit || !v {
		t.Fatal("pre-warmed entry should be visible through the VM")
	}
}
