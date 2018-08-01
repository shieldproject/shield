package core

var DefaultCompressionType = "bzip2"

func ValidCompressionType(t string) bool {
	return t == "bzip2" || t == "none"
}
