package parser

func RemoveSpaces(t PToken) bool {
	return t.Data.Flag("remove-spaces")
}
