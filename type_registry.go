package gad

type TypeRegistryKey uint16

type TypeRegistry struct {
	newKey TypeRegistryKey
	types  map[TypeRegistryKey]ObjectType
}

func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{types: make(map[TypeRegistryKey]ObjectType)}
}

func (st *TypeRegistry) Add(objectType ObjectType) TypeRegistryKey {
	st.types[st.newKey] = objectType
	st.newKey++
	return st.newKey
}

func (st *TypeRegistry) Get(key TypeRegistryKey) ObjectType {
	return st.types[key]
}

func (st TypeRegistry) Clone() *TypeRegistry {
	dst := make(map[TypeRegistryKey]ObjectType, len(st.types))
	for k, v := range st.types {
		dst[k] = v
	}
	st.types = dst
	return &st
}

var DefaultTypeRegistry = NewTypeRegistry()

func RegisterType(objectType ObjectType) TypeRegistryKey {
	return DefaultTypeRegistry.Add(objectType)
}
