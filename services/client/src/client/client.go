package client

import (
	"net"
	"time"
	"os"
	"fmt"
	"bufio"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200

const ECHO_CLIENT_BUFFER_SIZE = 512
const ECHO_CLIENT_MESSAGE_AMOUNT = 3
const ECHO_CLIENT_MESSAGE_DELAY_MS = 1

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile	 string
	OutputFile string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
	inputFile *os.File
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

	outputFile,err := os.Create(config.OutputFile)
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
	for i := range CONNECTION_ATTEMPTS_MAX {
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

	messageId := 0
	scanner := bufio.NewScanner(client.inputFile)
	for scanner.Scan() {

		messageArgs := []any{"agency-id", client.config.AgencyId, "message-id", messageId}
		logger.Info(mainAction, logger.InProgress, messageArgs...)
		clientMessage := scanner.Text() //aca obtengo la linea

		if err := safe_socket.SendAll(client.conn, []byte(clientMessage)); err != nil {
			logger.Error("send-message", logger.Fail, messageArgs...)
			return err
		}

		responseBuffer, err := safe_socket.RecvAll(client.conn, ECHO_CLIENT_BUFFER_SIZE)
		if err != nil {
			logger.Error("recv-response", logger.Fail, messageArgs...)
			return err
		}


		stringResponseBuffer := string(responseBuffer)

		_, err = fmt.Fprintln(client.outputFile, stringResponseBuffer)

		if err != nil {
			logger.Error("output-file-write", logger.Fail, messageArgs...)
			return err
		}

		if stringResponseBuffer != clientMessage {
			logger.Error("check-response", logger.Fail, messageArgs...)
			return err
		}

		time.Sleep(ECHO_CLIENT_MESSAGE_DELAY_MS * time.Millisecond)

		messageId++
	}
	if err := scanner.Err(); err != nil {
			// manejar error
	}


	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
	/*
	for messageId := range ECHO_CLIENT_MESSAGE_AMOUNT {
		messageArgs := []any{"agency-id", client.config.AgencyId, "message-id", messageId}
		logger.Info(mainAction, logger.InProgress, messageArgs...)

		clientMessage := client.config.AgencyId

		if err := safe_socket.SendAll(client.conn, []byte(clientMessage)); err != nil {
			logger.Error("send-message", logger.Fail, messageArgs...)
			return err
		}

		responseBuffer, err := safe_socket.RecvAll(client.conn, ECHO_CLIENT_BUFFER_SIZE)
		if err != nil {
			logger.Error("recv-response", logger.Fail, messageArgs...)
			return err
		}

		if string(responseBuffer) != clientMessage {
			logger.Error("check-response", logger.Fail, messageArgs...)
			return err
		}

		time.Sleep(ECHO_CLIENT_MESSAGE_DELAY_MS * time.Millisecond)
	}
	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)

	return nil

	*/
}


func (client *Client) Close() error {
	client.conn.Close()
	client.inputFile.Close()
	client.outputFile.Close()
	return nil
}
