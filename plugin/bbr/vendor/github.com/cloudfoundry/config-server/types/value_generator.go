package types

type ValueGenerator interface {
	Generate(interface{}) (interface{}, error)
}
