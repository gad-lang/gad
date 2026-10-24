package gad

// addTypeCtors registers one typed single-argument constructor method per
// accepted input type on a builtin object type. Each method delegates to the
// type's existing constructor (which type-switches on the argument), so the
// only effect is that the typed `T(v <kind>)` headers appear in the type's
// methods (e.g. in `repr(T; indent)`). The default constructor still handles
// any input not matched by a typed method.
func addTypeCtors(typ *BuiltinObjType, name string, ctor func(Call) (Object, error), paramTypes ...ObjectType) {
	for _, pt := range paramTypes {
		pt := pt
		AddMethod(typ, NewFunction(name, ctor,
			FunctionWithParams(func(p func(name string) *ParamBuilder) {
				p("v").Type(pt)
			}),
		))
	}
}

func init() {
	addTypeCtors(TimeType, "time", NewTimeFunc,
		TimeType, TStr, TRawStr, CalendarDateType, CalendarDateTimeType, TInt, TUint)
	addTypeCtors(CalendarDateType, "cdate", NewCalendarDateFunc,
		CalendarDateType, CalendarDateTimeType, TimeType, TUint, TInt, TStr)
	addTypeCtors(CalendarDateTimeType, "cdtime", NewCalendarDateTimeFunc,
		CalendarDateTimeType, TimeType, CalendarDateType, TUint, TInt, TStr)
	addTypeCtors(TimeOfDayType, "ctime", NewTimeOfDayFunc,
		TimeOfDayType, DurationType, CalendarDateTimeType, TimeType, TStr, TRawStr)
	addTypeCtors(DurationType, "duration", NewDurationFunc,
		DurationType, TInt, TUint, TStr)
	addTypeCtors(TimeLocationType, "Location", NewLocationFunc,
		TimeLocationType, TStr, TRawStr, TInt)
}
