package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// An enum reflects as a class does — @name, @fullName, @module, @items —,
// and so does a member — @name, @value, @index, @enum —, the plain keys of
// a member still answering.
func TestEnumReflect(t *testing.T) {
	testExpectRun(t, `
		enum Room { Kitchen, Bath = 5, Garage }
		return [
			Room.@name,
			Room.@fullName,
			typeName(Room.@module),
			[[m.@name, m.@value, m.@index] for m in Room.@items],
			Room.Bath.@enum == Room,
			[Room.Garage.name, Room.Garage.value, Room.Garage.index],
		]`,
		nil, Array{
			Str("Room"), Str("(main).Room"), Str("Module"),
			Array{Array{Str("Kitchen"), Uint(1), Int(0)}, Array{Str("Bath"), Int(5), Int(1)}, Array{Str("Garage"), Int(6), Int(2)}},
			True, Array{Str("Garage"), Int(6), Int(2)},
		})
}
