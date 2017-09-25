package fakes

type FakeSha1Calculator struct {
	calculateInputs map[string]CalculateInput
}

func NewFakeSha1Calculator() *FakeSha1Calculator {
	return &FakeSha1Calculator{}
}

type CalculateInput struct {
	Sha1 string
	Err  error
}

func (c *FakeSha1Calculator) Calculate(path string) (string, error) {
	calculateInput := c.calculateInputs[path]
	return calculateInput.Sha1, calculateInput.Err
}

func (c *FakeSha1Calculator) SetCalculateBehavior(calculateInputs map[string]CalculateInput) {
	c.calculateInputs = calculateInputs
}
