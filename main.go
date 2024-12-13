package main

import (
	"flag"
	"fmt"
	"github.com/tarm/serial"
	"io"
	"net"
	"os"
	"time"
)

const (
	HOST       = "127.0.0.1"
	PORT       = "5600"
	TYPE       = "tcp"
	BufferSize = 1024
)

// Определение активности COM порта
var comIsActive bool = false

// Определение закрытия соединения
var tcpIsClosed bool = true

func main() {
	// Определение флагов командной строки
	parComPort := flag.String("port", "COM4", "COM порт (обязательное)")
	argComBaud := flag.Int("baud", 57600, "Baud rate COM порта")
	flag.Parse()

	// Проверка обязательного параметра
	if *parComPort == "" {
		flag.PrintDefaults()
		panic("Обязательно указание COM порта")
	}

	comPort := *parComPort
	comBaud := *argComBaud

	// TCP сервер
	addr := HOST + ":" + PORT
	listener, err := net.Listen(TYPE, addr)
	if err != nil {
		fmt.Println("Ошибка создания TCP сервера: ", err)
		os.Exit(1)
	}
	defer listener.Close()

newConnection:
	for {
		fmt.Println("TCP сервер ожидает подключений на ", addr)
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка установления соединения:", err.Error())
			continue
		}
		fmt.Println("Установлено соединение с:", conn.RemoteAddr().String())
		tcpIsClosed = false

	readWrite:
		// Работа с COM и TCP
		for {
			port := comConnect(comPort, comBaud)
			defer port.Close()

			if port == nil {
				fmt.Printf("Ошибка! Порт %s не доступен...\n", comPort)
				time.Sleep(3 * time.Second)
				continue
			} else {
				fmt.Printf("Подключено к %s\n", comPort)
				comIsActive = true
			}

			// Отправка команд
			go readTcpWriteCom(conn, port)

			// Работа с потоком телеметрии
			res := readComWriteTcp(port, conn)
			switch res {
			case -1:
				port.Close()
				continue readWrite
			case -2:
				port.Close()
				conn.Close()
				continue newConnection
			}
		}
	}

}

func comConnect(comPort string, comBaud int) *serial.Port {
	config := &serial.Config{
		Name: comPort,
		Baud: comBaud,
	}

	port, err := serial.OpenPort(config)
	if err != nil {
		return nil
	}

	return port
}

// Прием команд по TCP и отправка их по COM
func readTcpWriteCom(tcpConn net.Conn, comPort *serial.Port) {
	buffer := make([]byte, BufferSize)

	for {
		if tcpIsClosed || !comIsActive {
			break
		}

		n, err := tcpConn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				continue
			}

			fmt.Println("Ошибка получения данных по TCP:", err)
			tcpIsClosed = true
			break
		}
		//fmt.Printf("TCP bytes received: %d\n", n)

		if n > 0 {
			_, err = comPort.Write(buffer[:n])
			if err != nil {
				fmt.Println("Ошибка записи в COM:", err)
				comIsActive = false
				break
			}
		}
	}
}

// Чтение из COM порта и отправка по TCP
func readComWriteTcp(comPort *serial.Port, tcpConn net.Conn) int8 {
	buffer := make([]byte, BufferSize)

	for {
		if tcpIsClosed {
			return -2
		}
		if !comIsActive {
			return -1
		}

		n, err := comPort.Read(buffer)
		if err != nil {
			fmt.Println("Ошибка чтения по COM:", err)
			comIsActive = false
			return -1
		}
		//fmt.Printf("Bytes received: %d\nData: %v\n", n, buffer[:n])

		if n > 0 {
			_, err = tcpConn.Write(buffer[:n])
			if err != nil {
				fmt.Println("Ошибка передачи по TCP:", err)
				tcpIsClosed = true
				return -2
			}
		}
	}
}
