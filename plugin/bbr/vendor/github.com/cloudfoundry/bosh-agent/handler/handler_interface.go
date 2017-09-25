package handler

type Func func(req Request) (resp Response)

type Handler interface {
	Run(handlerFunc Func) error
	Start(handlerFunc Func) error
	Stop()

	RegisterAdditionalFunc(handlerFunc Func)

	Send(target Target, topic Topic, message interface{}) error
}
