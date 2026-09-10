import os
import socket
from enum import IntEnum

import safe_socket
from lottery import Bet

MAGIC = 2777666555

class MessageType(IntEnum):
    BET = 0
    STOP = 1
    BAT = 2

class Stop:
    def __init__(self, agency_id: int):
        self.agency_id = agency_id

def _bytes_to_uint32_be(b: bytes) -> int:
    return (b[0] << 24) | (b[1] << 16) | (b[2] << 8) | b[3]


def _uint32_to_bytes_be(value: int, buf: bytearray, offset: int):
    buf[offset] = (value >> 24) & 0xFF
    buf[offset + 1] = (value >> 16) & 0xFF
    buf[offset + 2] = (value >> 8) & 0xFF
    buf[offset + 3] = value & 0xFF


def deserialize(sock: socket.socket):
    header = safe_socket.recv_all(sock, 10)
    if _bytes_to_uint32_be(header[:4]) != MAGIC:
        raise RuntimeError("bad magic")
    message_type = MessageType(header[4])
    agency_id = header[5]
    payload_len = _bytes_to_uint32_be(header[6:10])
    payload = safe_socket.recv_all(sock, payload_len)
    if message_type == MessageType.STOP:
        return Stop(agency_id)
    if message_type == MessageType.BET:
        fields = payload.decode("utf-8").split(",")
        if len(fields) != 5:
            raise RuntimeError("invalid bet payload")
        name, surname, document, birthdate, number = fields
        return Bet(agency_id, name, surname, int(document), birthdate, int(number))
    if message_type == MessageType.BAT:
        bets = []
        lines = payload.decode("utf-8").split("\n")
        for line in lines:
            fields = line.split(",")
            if len(fields) != 5:
                raise RuntimeError("invalid bat payload")
            name, surname, document, birthdate, number = fields
            bet = Bet(agency_id, name, surname, int(document), birthdate, int(number))
            bets.append(bet)
        return bets

    raise RuntimeError("unknown message type")


def serialize(sock: socket.socket, message_type: MessageType, bet_s):
    payload = b""
    if message_type == MessageType.BET:
        payload = f"{bet_s.first_name},{bet_s.last_name},{bet_s.document},{bet_s.birthdate},{bet_s.number}".encode()

    elif message_type == MessageType.BAT:
        agency_id = bet_s[0].agency_id
        payload = "\n".join(
            f"{item.first_name},{item.last_name},{item.document},{item.birthdate},{item.number}"
            for item in bet_s
        ).encode()

    agency_id = (bet_s[0].agency_id if message_type == MessageType.BAT
                 else bet_s.agency_id if bet_s is not None else 0)
    if not 0 <= agency_id <= 255:
        raise ValueError("agency id must fit in one byte")
    payload_len = len(payload)
    buf = bytearray(10 + payload_len)
    _uint32_to_bytes_be(MAGIC, buf, 0)
    buf[4] = message_type
    buf[5] = agency_id
    _uint32_to_bytes_be(payload_len, buf, 6)
    buf[10:] = payload
    safe_socket.send_all(sock, bytes(buf))
