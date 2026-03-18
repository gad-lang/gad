package gad

type StaticTypeKey uint16

type StaticTypes struct {
	newKey StaticTypeKey
	types  map[StaticTypeKey]ObjectType
}

func NewStaticTypes() *StaticTypes {
	return &StaticTypes{types: make(map[StaticTypeKey]ObjectType)}
}

func (st *StaticTypes) Add(objectType ObjectType) StaticTypeKey {
	st.types[st.newKey] = objectType
	st.newKey++
	return st.newKey
}

func (st *StaticTypes) Get(key StaticTypeKey) ObjectType {
	return st.types[key]
}

func (st StaticTypes) Clone() *StaticTypes {
	dst := make(map[StaticTypeKey]ObjectType, len(st.types))
	for k, v := range st.types {
		dst[k] = v
	}
	st.types = dst
	return &st
}

var DefaultStaticTypes = NewStaticTypes()

func NewStaticType(objectType ObjectType) StaticTypeKey {
	return DefaultStaticTypes.Add(objectType)
}
