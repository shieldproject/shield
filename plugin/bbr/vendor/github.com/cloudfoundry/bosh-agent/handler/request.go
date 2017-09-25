package handler

type ProtocolVersion int

func NewRequest(replyTo, method string, payload []byte, protocolVersion ProtocolVersion) Request {
	return Request{
		ReplyTo:         replyTo,
		Method:          method,
		Payload:         payload,
		ProtocolVersion: protocolVersion,
	}
}

type Request struct {
	ReplyTo         string `json:"reply_to"`
	Method          string
	Payload         []byte
	ProtocolVersion ProtocolVersion `json:"protocol"`
}

func (r Request) GetPayload() []byte {
	return r.Payload
}
