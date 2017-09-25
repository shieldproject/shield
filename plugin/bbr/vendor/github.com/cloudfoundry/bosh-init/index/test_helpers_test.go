package index_test

type Key struct {
	Key string
}

type Value struct {
	Name  string
	Count float64
}

type ArrayValue struct{ Names []string }

type StructValue struct{ Name Name }

type Name struct {
	First  string
	Middle *string
	Last   string
}
