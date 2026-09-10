import signal
import threading

class GracefulShutdown:
    def __init__(self):
        self.event = threading.Event()
        signal.signal(signal.SIGTERM, self._handle)
    def _handle(self, signum, frame):
        self.event.set()

class ShutdownRequested(Exception):
    pass