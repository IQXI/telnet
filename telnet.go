package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/spf13/pflag"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var port string
var host string

func init() {
	pflag.StringVarP(&port, "port", "p", "23", "port number")
	pflag.StringVarP(&host, "host", "h", "localhost", "host name or ip address")
}

func readThread(ctx context.Context, conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for {
		select {
		case <-ctx.Done():
			break
		default:
			if !scanner.Scan() {
				//если не можем прочитать, то останавливаем программу
				log.Fatalf("Connection closed by remote %v", conn.RemoteAddr())
			}
			text := scanner.Text()
			log.Printf("%v -> %v : %s", conn.RemoteAddr(), "You", text)
		}
	}
}

func writeThread(ctx context.Context, conn net.Conn) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			break
		default:
			if !scanner.Scan() {
				log.Printf("Cannot scan from StdIn.")
			}
			text := scanner.Text()
			log.Printf("%v -> %v : %s", "You", conn.RemoteAddr(), text)

			_, err := conn.Write([]byte(fmt.Sprintf("%s\n", text)))
			if err != nil {
				log.Fatalf("Cannot write to %v: %v", conn.RemoteAddr(), err)
			}
		}

	}
}

func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\rCtrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

func main() {

	pflag.Parse()
	SetupCloseHandler()

	dialer := &net.Dialer{}

	//родительский пустой контекст
	ctx := context.Background()
	//делаем контекст с тайм аутом
	ctx, _ = context.WithTimeout(ctx, 30*time.Second)

	//пытаемся подключиться
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%v:%v", host, port))

	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}

	defer conn.Close()

	log.Printf("Connected to host: %v port: %v", host, port)

	//создаем вейт группу для запуска наших горутин и их бесконечного выполнения
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		readThread(ctx, conn)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		writeThread(ctx, conn)
		wg.Done()
	}()
	wg.Wait()
}
