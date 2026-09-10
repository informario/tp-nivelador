package safe_socket

import (
	"context"
	"io"
)

//TODO: Complete with a short-read/short-write tolerant implementation

func SendAll(socket io.Writer, bytes []byte) error {
	/*
		Acá no ocurre lo mismo como con el Reader, no hay un
		"Write conventionally writes what is available instead
		of waiting for more.". En todo caso donde Writer.Write
		devuelve n < P y un error, debo propagar eso hacia el
		caller

		TODO Cambiar comentario, que esta mal
	*/
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
	/*
		Acá no ocurre lo mismo como con el Reader, no hay un
		"Write conventionally writes what is available instead
		of waiting for more.". En todo caso donde Writer.Write
		devuelve n < P y un error, debo propagar eso hacia el
		caller

		TODO Cambiar comentario, que esta mal
	*/
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
	/*
		Primero proceso los bytes antes de evaluar el error
	*/
	buff := make([]byte, size)
	contador := 0
	for contador < size {
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
