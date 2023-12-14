package websocket

type Opcode int

const (
	Unknown Opcode = -1
	Text           = 0
	Binary         = 1
	Close          = 8
	Ping           = 9
	Pong           = 10
)

func IsFinished(frame uint8) bool {
	return frame>>7 == 1
}

func GetOpcode(frame uint8) Opcode {
	switch frame & 0b1111 {
	case 0:
		return Text
	case 1:
		return Binary
	case 8:
		return Close
	case 9:
		return Ping
	case 10:
		return Pong
	}

	return Unknown
}
