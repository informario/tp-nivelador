import csv
import io
import os
from collections.abc import Iterator
from .bet import Bet

_LOTTERY_WINNER_NUMBER = 7574


class Lottery:
    def __init__(self, storage_path) -> None:
        self.storage_path = storage_path

    def has_won(self, bet: Bet) -> bool:
        return bet.number == _LOTTERY_WINNER_NUMBER

    def store_bets(self, bets: list[Bet]) -> None:
        with open(self.storage_path, "ab") as file:
            for bet in bets:
                row = io.StringIO(newline="")
                csv.writer(row, quoting=csv.QUOTE_MINIMAL).writerow([
                    bet.agency_id, bet.first_name, bet.last_name, bet.document,
                    bet.birthdate, bet.number,
                ])
                data = row.getvalue().encode("utf-8")
                if file.write(data) != len(data):
                    raise OSError("bad write file")

    def load_bets(self) -> Iterator[Bet]:
        with open(self.storage_path, "rb") as file:
            expected_bytes = os.fstat(file.fileno()).st_size
            data = file.read(expected_bytes)
            if len(data) != expected_bytes:
                raise OSError("bad read file")

        reader = csv.reader(
            io.StringIO(data.decode("utf-8"), newline=""),
            quoting=csv.QUOTE_MINIMAL,
        )
        for row in reader:
            [agency_id, first_name, last_name, document, birthdate, number] = row
            yield Bet(
                int(agency_id), first_name, last_name, int(document), birthdate,
                int(number),
            )
