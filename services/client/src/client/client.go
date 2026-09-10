package client

import (
	"bufio"
	"context"
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

func (client *Client) Run(ctx context.Context) error {
	const mainAction = "test-echo-server"
	if err := ctx.Err(); err != nil {
		return err
	}
	//ctx no puede cancelar un socket por si mismo
	stopCloseOnCancel := make(chan struct{})
	defer close(stopCloseOnCancel)
	go func() {
		select {
		case <-ctx.Done():
			_ = client.conn.Close()
		case <-stopCloseOnCancel:
		}
	}()

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
		//al leer linea x linea, no debería escalar el consumo de memoria aumentando la entrada
		if ctx.Err() != nil {
			return ctx.Err()
		}
		batch = append(batch, scanner.Text())
		if len(batch) == batchSize {
			err := client.sendBatch(batch, byte(agencyID), ctx)
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
		if err := client.sendBatch(batch, byte(agencyID), ctx); err != nil {
			return err
		}
	}
	if err := client.sendWithRetry(nil, byte(agencyID), lottery.MESSAGE_STOP, ctx); err != nil {
		logger.Error("stop-message", logger.Fail)
		return err
	}
	if err := client.receiveWinners(ctx); err != nil {
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

func (client *Client) sendBatch(batch []string, agencyID byte, ctx context.Context) error {
	payload := strings.Join(batch, "\n")
	//este sleep es para no sobrecargar los sockets y que no se dispare el consumo de memoria
	//tambien esto puede ser corregido con un ack, pero esta es otra forma
	if err := waitForContext(ctx, time.Millisecond); err != nil {
		return err
	}
	return client.sendWithRetry(&payload, agencyID, lottery.MESSAGE_BAT, ctx)
}

func (client *Client) receiveWinners(ctx context.Context) error {
	for {
		messageType, bet, _, err := lottery.Deserialize(client.conn, ctx)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
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

func (client *Client) sendWithRetry(bet *string, agencyID byte, msgType lottery.MessageType, ctx context.Context) error {
	/*Retry 3 veces por cada mensaje*/
	/*si no pude enviar un mesaje 3 veces, en el caller siempre lanzo una excepción*/
	/*El reintento también debe poder interrumpirse por SIGTERM.*/
	var err error
	for attempt := 0; attempt < RETIRES_ATTEMPTS_MAX; attempt++ {
		var lotteryBet *lottery.Bet
		if bet != nil {
			value := lottery.Bet(*bet)
			lotteryBet = &value
		}
		err = lottery.Serialize(lotteryBet, agencyID, client.conn, msgType, ctx)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if err := waitForContext(ctx, RETIRES_ATTEMPS_DELAY_MS*time.Millisecond); err != nil {
			return err
		}
	}
	return fmt.Errorf("max retries exceeded: %w", err)
}

func waitForContext(ctx context.Context, duration time.Duration) error {
	//esto es basicamente un sleep pero que contempla el shutdown
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (client *Client) Close() error {
	/*Esto me resuelve el cierre de los FD, asumo que no me interesan mensajes de error de Close
	dado que si recibo eso, qué puedo hacer¿?*/
	return errors.Join(
		client.conn.Close(),
		client.inputFile.Close(),
		client.outputFile.Close(),
	)
}
