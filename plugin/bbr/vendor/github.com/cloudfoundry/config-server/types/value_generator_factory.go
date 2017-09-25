package types

type ValueGeneratorFactory interface {
	GetGenerator(valueType string) (ValueGenerator, error)
}
