package lottery

import (
	"context"
	"encoding/binary"
	"errors"
	"io"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

type Bet string

type MessageType uint32

const (
	MESSAGE_BET  MessageType = 0
	MESSAGE_STOP MessageType = 1
	MESSAGE_BAT  MessageType = 2
)

const MAGIC = 2777666555

func bytesToUint32BE(data []byte) uint32 {
	return uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
}

/*
A partir de una línea del csv, cuento bytes totales y le agrego un magic, el tipo, el agencyid y el tamaño
para el mensaje bet, le agrego un payload

todo! deberia sacar esto de aca y ponerlo en un protocol.py
*/

func Serialize(bet *Bet, agencyID byte, sock io.Writer, messageType MessageType, ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var payload []byte
	if messageType == MESSAGE_BET || messageType == MESSAGE_BAT {
		payload = []byte(*bet)
	}
	payloadLen := uint32(len(payload))
	buf := make([]byte, 10+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], MAGIC)
	buf[4] = byte(messageType)
	buf[5] = agencyID
	binary.BigEndian.PutUint32(buf[6:10], payloadLen)
	copy(buf[10:], payload)
	if err := safe_socket.SendAll2(sock, buf, ctx); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return err
	}
	return nil
}

func Deserialize(sock io.Reader, ctx context.Context) (MessageType, *Bet, byte, error) {
	header, err := safe_socket.RecvAll2(sock, 10, ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, nil, 0, ctxErr
		}
		return 0, nil, 0, err
	}
	if bytesToUint32BE(header[:4]) != MAGIC {
		return 0, nil, 0, errors.New("bad magic")
	}
	messageType := MessageType(header[4])
	agencyID := header[5]
	payloadLen := bytesToUint32BE(header[6:10])
	payload, err := safe_socket.RecvAll2(sock, int(payloadLen), ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, nil, 0, ctxErr
		}
		return 0, nil, 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, nil, 0, err
	}
	switch messageType {
	case MESSAGE_STOP:
		return MESSAGE_STOP, nil, agencyID, nil
	case MESSAGE_BET:
		bet := Bet(payload)
		return MESSAGE_BET, &bet, agencyID, nil
	case MESSAGE_BAT:
		bet := Bet(payload)
		return MESSAGE_BAT, &bet, agencyID, nil
	default:
		return 0, nil, 0, errors.New("unknown message type")
	}
}
