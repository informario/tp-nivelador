package safe_socket

import (
	"context"
	"io"
)

/*
Tengo 2 implementaciones, una que uso yo, al que le paso el contexto para el graceful shutdown, y otra para los tests de shrot read/write

La lógica es la misma, salvo el agregado del graceful shutdown

Esto porque go no adopta sobrecarga
*/

func SendAll(socket io.Writer, bytes []byte) error {
	size := len(bytes)
	for contador := 0; contador < size; {
		n, err := socket.Write(bytes[contador:])
		contador += n
		if err != nil {
			return err
		}
	}
	return nil
}
func SendAll2(socket io.Writer, bytes []byte, ctx context.Context) error {

	size := len(bytes)
	for contador := 0; contador < size; {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n, err := socket.Write(bytes[contador:])
		contador += n
		if err != nil {
			return err
		}
	}
	return nil
}

func RecvAll(socket io.Reader, size int) ([]byte, error) {
	buff := make([]byte, size)
	contador := 0
	for contador < size {
		n, err := socket.Read(buff[contador:])
		contador += n
		if err == io.EOF && contador < size {
			if contador == 0 {
				return buff[:0], io.EOF
			}
			return buff[:contador], io.ErrUnexpectedEOF
		} else if err != nil && err != io.EOF {
			return buff[:contador], err
		}
	}
	return buff[:contador], nil
}
func RecvAll2(socket io.Reader, size int, ctx context.Context) ([]byte, error) {
	/*
		Primero proceso los bytes antes de evaluar el error
	*/
	buff := make([]byte, size)
	contador := 0
	for contador < size {
		if ctx.Err() != nil {
			return buff, ctx.Err()
		}
		n, err := socket.Read(buff[contador:])
		contador += n
		/*
			EOF lo tengo q transformar a UnexpectedEOF si el contador < size
			EOF lo transformo en nil si contador no es < size
			Cualquier otro error, lo propago
		*/
		if err == io.EOF && contador < size {
			if contador == 0 {
				return buff[:0], io.EOF
			}
			return buff[:contador], io.ErrUnexpectedEOF
		} else if err != nil && err != io.EOF {
			return buff[:contador], err
		}
	}
	return buff[:contador], nil
}
