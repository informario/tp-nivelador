import socket as socket_module
MAGIC = 2777666555
try:
    from ..shutdown import GracefulShutdown, ShutdownRequested
except ImportError:
    from shutdown import GracefulShutdown, ShutdownRequested

TIMEOUT_CORTO = 0.2


def recv_all(sock: socket_module.socket, size, shutdown: GracefulShutdown | None = None):
    buffs = []
    contador = 0
    if shutdown is not None:
        sock.settimeout(TIMEOUT_CORTO)
    while contador < size:
        try:
            chunk = sock.recv(size - contador)
        except socket_module.timeout as exc:
            if shutdown is not None and shutdown.event.is_set():
                raise ShutdownRequested() from exc
            continue
        if chunk == b'':
            raise RuntimeError("Connection closed")
        buffs.append(chunk)
        contador += len(chunk)
    return b''.join(buffs)



def send_all(sock: socket_module.socket, bytes, shutdown: GracefulShutdown | None = None):
    contador = 0
    if shutdown is not None:
        sock.settimeout(TIMEOUT_CORTO)
    while contador < len(bytes):
        try:
            n = sock.send(bytes[contador:])
        except socket_module.timeout as exc:
            if shutdown is not None and shutdown.event.is_set():
                raise ShutdownRequested() from exc
            continue
        except socket_module.error as exc:
            raise RuntimeError("Connection closed") from exc
        contador += n
    return contador
