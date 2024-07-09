package parser

func RemoveSpaces(t Token) bool {
	return t.Data.Flag("remove-spaces")
}
