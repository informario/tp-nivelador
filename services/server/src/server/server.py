import socket
import threading
import os
import logger
import lottery
import protocol

_ECHO_SERVER_MESSAGE_SIZE = 1024


class Server:
    def __init__(self, server_host: str, server_port: int) -> None:
        self.server_host = server_host
        self.server_port = server_port
        quorum_min = int(os.environ.get("AGENCY_QUORUM_MIN", "5"))
        self._barrier = threading.Barrier(quorum_min)

    def _handle_client(self, client_socket):
        action = "handle-client"
        message_amount = 0
        bets = []
        client_agency_id = None
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                message = protocol.deserialize(client_socket)
                if isinstance(message, lottery.Bet):
                    bets.append(message)
                    lottery.Lottery("store.csv").store_bets([message])
                    message_amount += 1
                    continue
                if isinstance(message, list):
                    bets.extend(message)
                    lottery.Lottery("store.csv").store_bets(message)
                    message_amount += len(message)
                    continue
                if isinstance(message, protocol.Stop):
                    client_agency_id = message.agency_id
                    break
            logger.info(
                action,
                logger.LogResult.in_progress,
                "waiting-at-barrier", self._barrier.n_waiting + 1,
                "barrier-size", self._barrier.parties,
            )
            self._barrier.wait()

            result = lottery.Lottery("store.csv")
            winners = [bet for bet in result.load_bets() if result.has_won(bet)]
            winning_bets = {
                (bet.agency_id, bet.first_name, bet.last_name, bet.document,
                 bet.birthdate, bet.number)
                for bet in winners
            }
            for bet in bets:
                if (bet.agency_id == client_agency_id and
                        (bet.agency_id, bet.first_name, bet.last_name, bet.document,
                        bet.birthdate, bet.number) in winning_bets):
                    protocol.serialize(client_socket, protocol.MessageType.BET, bet)

            logger.info(action, logger.LogResult.success,
                        "messages-amount", message_amount)
        except threading.BrokenBarrierError:
            logger.error(action, logger.LogResult.fail,
                         "messages-amount", message_amount)
        except Exception:
            logger.error(action, logger.LogResult.fail,
                         "messages-amount", message_amount)
            raise
        finally:
            client_socket.close()
            logger.info(
                action,
                logger.LogResult.in_progress,
                "clients closed",
            )

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                client_thread = threading.Thread(
                    target=self._handle_client,
                    args=(client_socket,),
                    daemon=True,
                )
                client_thread.start()
