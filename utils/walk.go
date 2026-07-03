package utils

type WalkMode uint8

const (
	WalkModeContinue WalkMode = iota
	WalkModeBreak
	WalkModeSkipSiblings
	WalkModeSkipChildren
)

func Walk[T any](e T, each func(e T, cb func(T) WalkMode) bool, cb func(path []T, e T) (mode WalkMode)) {
	var walk func(path []T, e T) WalkMode

	walk = func(path []T, e T) (mode WalkMode) {
		mode = cb(path, e)
		switch mode {
		case WalkModeBreak, WalkModeSkipSiblings:
			return
		case WalkModeSkipChildren:
			return WalkModeContinue
		}

		path = append(path, e)

		if !each(e, func(child T) WalkMode {
			return walk(path, child)
		}) {
			return WalkModeBreak
		}

		return WalkModeContinue
	}

	each(e, func(e T) WalkMode {
		return walk(nil, e)
	})
}

func SingleWalk[T any](e T, each func(e T, cb func(T) WalkMode) bool, cb func(e T) (mode WalkMode)) {
	var walk func(e T) WalkMode

	walk = func(e T) (mode WalkMode) {
		mode = cb(e)
		switch mode {
		case WalkModeBreak, WalkModeSkipSiblings:
			return
		case WalkModeSkipChildren:
			return WalkModeContinue
		}

		if !each(e, func(child T) WalkMode {
			return walk(child)
		}) {
			return WalkModeBreak
		}

		return WalkModeContinue
	}

	each(e, func(e T) WalkMode {
		return walk(e)
	})
}
