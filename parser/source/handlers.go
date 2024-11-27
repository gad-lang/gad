package source

type NextHandlers struct {
	LineEndHandlers     []func()
	PostLineEndHandlers []func()
	EOFHandlers         []func(r *Reader)
}

func (h *NextHandlers) LineEndHandler(f func()) {
	h.LineEndHandlers = append(h.LineEndHandlers, f)
}

func (h *NextHandlers) CallLineEndHandlers() {
	for _, handler := range h.LineEndHandlers {
		handler()
	}
	h.LineEndHandlers = nil
}

func (h *NextHandlers) PostLineEndHandler(f func()) {
	h.PostLineEndHandlers = append(h.PostLineEndHandlers, f)
}

func (h *NextHandlers) CallPostLineEndHandlers() {
	for _, handler := range h.PostLineEndHandlers {
		handler()
	}
	h.PostLineEndHandlers = nil
}

func (h *NextHandlers) EOFHandler(f func(r *Reader)) {
	h.EOFHandlers = append(h.EOFHandlers, f)
}

func (s *Reader) CallEOFHandlers() {
	handlers := s.EOFHandlers
	s.EOFHandlers = nil

	for _, handler := range handlers {
		handler(s)
	}
}
