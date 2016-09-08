package test

func MakeInt(value int) *int {
	v := new(int)
	*v = value
	return v
}

func MakeString(value string) *string {
	v := new(string)
	*v = value
	return v
}
