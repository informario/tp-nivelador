import os
import sys
import logger
import server

from shutdown import GracefulShutdown

SERVER_HOST = os.environ["SERVER_HOST"]
SERVER_PORT = int(os.environ["SERVER_PORT"])

def main():
    shutdown = GracefulShutdown()
    logger.init()

    s = server.Server(SERVER_HOST, SERVER_PORT, shutdown)
    try:
        s.run()
    except Exception as e:
        logger.error("server-run", logger.LogResult.fail, "err", e)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
