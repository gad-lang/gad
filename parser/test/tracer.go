package test

type Tracer struct {
	Out []string
}

func (o *Tracer) Write(p []byte) (n int, err error) {
	o.Out = append(o.Out, string(p))
	return len(p), nil
}
