import socket
MAGIC = 2777666555

# TODO: Complete with a short-read/short-write tolerant implementation


def recv_all(socket: socket.socket, size):
    buffs = []
    contador = 0
    while contador < size:
        buffs.append(socket.recv(size - contador))
        if buffs[-1] == b'':
            raise RuntimeError("Connection closed")
        contador = contador + len(buffs[-1])
    return b''.join(buffs)


def send_all(socket: socket.socket, bytes):
    contador = 0
    while contador < len(bytes):
        try:
            n = socket.send(bytes[contador:])
        except socket.error as e:
            raise RuntimeError("Connection closed") from e
        contador = contador + n
    return contador
