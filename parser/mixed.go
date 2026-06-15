package parser

func RemoveSpaces(t PToken) bool {
	return t.Data.Flag("remove-spaces")
}

// RemoveAllSpaces reports the double-dash trim marker (`{%--` / `--%}`) on a
// mixed delimiter token, which strips ALL adjacent whitespace.
func RemoveAllSpaces(t PToken) bool {
	return t.Data.Flag("remove-spaces-all")
}
