package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/tarm/serial"
	"gopkg.in/webnice/pdu.v1"
)

var (
	tel     string
	smsText string
	comPort string
)

func init() {
	flag.StringVar(&tel, "t", "", "telephone number")
	flag.StringVar(&smsText, "sms", "", "sms text")
	flag.StringVar(&comPort, "port", "COM1", "COM port name")
}
func main() {
	flag.Parse()
	if tel == "" || smsText == "" {
		fmt.Println("telephone and sms text is required")
		flag.PrintDefaults()
		return
	}

	port, err := serial.OpenPort(&serial.Config{
		Name:        comPort,
		Baud:        460800,
		Parity:      serial.ParityOdd,
		StopBits:    serial.Stop1,
		ReadTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = port.Flush(); err != nil {
			fmt.Println("Err(flush):", err)
		}
		if err = port.Close(); err != nil {
			fmt.Println("Err(close):", err)
		}
	}()

	_, err = port.Write([]byte("AT\r\n"))
	if err != nil {
		panic(err)
	}
	waitOK(port)

	_, err = port.Write([]byte("AT+CMGF=0\r\n"))
	if err != nil {
		panic(err)
	}
	waitOK(port)

	sms := pdu.Encode{
		Address: tel,
		Message: smsText,
	}

	pduCoder := pdu.New()
	cmds, err := pduCoder.Encoder(sms)
	if err != nil {
		panic(err)
	}
	for _, cmd := range cmds {
		parts := strings.Split(cmd, "\r\n")
		_, err = port.Write([]byte(parts[0] + "\r\n"))
		if err != nil {
			panic(err)
		}
		time.Sleep(500 * time.Millisecond)
		_, err = port.Write([]byte(parts[1] + "\x1A\r\n"))
		if err != nil {
			panic(err)
		}
		waitOK(port)
	}
}

func waitOK(p *serial.Port) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ch := make(chan struct{}, 1)
	defer close(ch)

	go func() {
		buf := make([]byte, 255)
		for {

			n, err := p.Read(buf)
			if err != nil {
				panic(err)
			}
			if n != 0 {
				if bytes.Contains(buf[:n], []byte("OK")) {
					ch <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-ticker.C:
		panic(fmt.Errorf("Operation timeout"))
	case <-ch:
		return
	}
}
