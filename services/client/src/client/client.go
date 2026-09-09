package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/lottery"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200
const RETIRES_ATTEMPTS_MAX = 3
const RETIRES_ATTEMPS_DELAY_MS = 200

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
}

type Client struct {
	conn       net.Conn
	config     ClientConfig
	inputFile  *os.File
	outputFile *os.File
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	inputFile, err := os.Open(config.InputFile)
	if err != nil {
		logger.Warn("open-input-file", logger.Fail)
		conn.Close()
		return nil, err
	}

	outputFile, err := os.Create(config.OutputFile)
	if err != nil {
		logger.Warn("create-output-file", logger.Fail)
		conn.Close()
		inputFile.Close()
		return nil, err
	}

	client := &Client{conn: conn, config: config, inputFile: inputFile, outputFile: outputFile}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn
	logger.Info(action, logger.InProgress)
	for i := 0; i < CONNECTION_ATTEMPTS_MAX; i++ {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}
		logger.Info(action, logger.Success)
		break
	}
	return conn, err
}

func (client *Client) Run() error {
	const mainAction = "test-echo-server"
	defer client.conn.Close()
	agencyID, err := strconv.ParseUint(client.config.AgencyId, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid agency id: %w", err)
	}
	scanner := bufio.NewScanner(client.inputFile)
	batchSize, err := batchSizeFromEnv()
	if err != nil {
		return err
	}
	batch := make([]string, 0, batchSize)
	for scanner.Scan() {
		batch = append(batch, scanner.Text())
		if len(batch) == batchSize {
			err := client.sendBatch(batch, byte(agencyID))
			if err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.New("file not found")
	}
	if len(batch) > 0 {
		if err := client.sendBatch(batch, byte(agencyID)); err != nil {
			return err
		}
	}
	if err := client.sendWithRetry(nil, byte(agencyID), lottery.MESSAGE_STOP); err != nil {
		logger.Error("stop-message", logger.Fail)
		return err
	}
	if err := client.receiveWinners(); err != nil {
		logger.Error("winner-messages", logger.Fail)
		return err
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}

func batchSizeFromEnv() (int, error) {
	size, err := strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil {
		return 0, fmt.Errorf("no BATCH_SIZE")
	}
	return size, nil
}

func (client *Client) sendBatch(batch []string, agencyID byte) error {
	payload := strings.Join(batch, "\n")
	time.Sleep(1 * time.Millisecond)
	return client.sendWithRetry(&payload, agencyID, lottery.MESSAGE_BAT)
}

func (client *Client) receiveWinners() error {
	for {
		messageType, bet, _, err := lottery.Deserialize(client.conn)
		isEOF := errors.Is(err, io.EOF)
		if isEOF {
			return nil
		}
		if err != nil {
			return err
		}
		if bet == nil || (messageType != lottery.MESSAGE_BET && messageType != lottery.MESSAGE_BAT) {
			return fmt.Errorf("unexpected message type: %d", messageType)
		}
		if messageType == lottery.MESSAGE_BET {
			_, writeErr := fmt.Fprintln(client.outputFile, string(*bet))
			if writeErr != nil {
				return writeErr
			}
			continue
		}
		for _, line := range strings.Split(string(*bet), "\n") {
			_, writeErr := fmt.Fprintln(client.outputFile, line)
			if writeErr != nil {
				return writeErr
			}
		}
	}
}

func (client *Client) sendWithRetry(bet *string, agencyID byte, msgType lottery.MessageType) error {
	/*Retry 3 veces por cada mensaje*/
	/*si no pude enviar un mesaje 3 veces, en el caller siempre lanzo una excepción*/
	var err error
	for attempt := 0; attempt < RETIRES_ATTEMPTS_MAX; attempt++ {
		var lotteryBet *lottery.Bet
		if bet != nil {
			value := lottery.Bet(*bet)
			lotteryBet = &value
		}
		err = lottery.Serialize(lotteryBet, agencyID, client.conn, msgType)
		if err == nil {
			return nil
		}
		time.Sleep(RETIRES_ATTEMPS_DELAY_MS * time.Millisecond)
	}
	return fmt.Errorf("max retries exceeded: %w", err)
}

func (client *Client) Close() error {
	/*Esto me resuelve todo, asumo que no me interesan mensajes de error de Close
	dado que si recibo eso, qué puedo hacer¿?*/
	err := client.conn.Close()
	if err != nil {
		return err
	}
	err = client.inputFile.Close()
	if err != nil {
		return err
	}
	err = client.outputFile.Close()
	if err != nil {
		return err
	}
	return nil
}
