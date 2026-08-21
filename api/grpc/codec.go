package grpcapi

import "fmt"

type rawMessage struct {
	Data []byte
}

func (m *rawMessage) Reset() {
	m.Data = m.Data[:0]
}

func (m *rawMessage) String() string {
	return string(m.Data)
}

func (m *rawMessage) ProtoMessage() {}

type rawCodec struct{}

func (rawCodec) Name() string {
	return "jsonraw"
}

func (rawCodec) Marshal(v any) ([]byte, error) {
	msg, ok := v.(*rawMessage)
	if !ok {
		return nil, fmt.Errorf("raw codec expects *rawMessage")
	}
	return msg.Data, nil
}

func (rawCodec) Unmarshal(data []byte, v any) error {
	msg, ok := v.(*rawMessage)
	if !ok {
		return fmt.Errorf("raw codec expects *rawMessage")
	}
	msg.Data = append(msg.Data[:0], data...)
	return nil
}
