package internal

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"
)

const defaultMaxClients = 10

// Server содержит настройки TCP-сервера.
type Server struct {
	port       string
	maxClients int
}

// NewServer проверяет настройки и создаёт сервер.
func NewServer(port string, maxClients int) (*Server, error) {
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf(
			"invalid port: %q (expected 1..65535)",
			port,
		)
	}

	if maxClients < 1 {
		return nil, fmt.Errorf(
			"invalid client limit: %d (expected at least 1)",
			maxClients,
		)
	}

	return &Server{
		port:       strconv.Itoa(portNumber),
		maxClients: maxClients,
	}, nil
}

// Run разбирает аргументы командной строки и запускает сервер.
func Run(args []string) error {
	if len(args) != 2 {
		program := "task2_server"
		if len(args) > 0 {
			program = args[0]
		}

		return fmt.Errorf("usage: %s <port>", program)
	}

	server, err := NewServer(args[1], defaultMaxClients)
	if err != nil {
		return err
	}

	return server.ListenAndServe()
}

// ListenAndServe принимает подключения и ограничивает
// количество одновременно обслуживаемых клиентов.
func (s *Server) ListenAndServe() error {
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return fmt.Errorf("create uploads directory: %w", err)
	}

	address := net.JoinHostPort("", s.port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	defer func() {
		if err = listener.Close(); err != nil {
			log.Printf("close listener: %v", err)
		}
	}()

	slots := make(chan struct{}, s.maxClients)

	log.Printf(
		"server listening on %s; max clients=%d",
		listener.Addr(),
		s.maxClients,
	)

	var retryDelay time.Duration

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Закрытый listener не нужно пытаться использовать снова.
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			var netErr net.Error
			if errors.As(err, &netErr) &&
				(netErr.Timeout() || netErr.Temporary()) {

				if retryDelay == 0 {
					retryDelay = 5 * time.Millisecond
				} else {
					retryDelay *= 2
				}

				if retryDelay > time.Second {
					retryDelay = time.Second
				}

				log.Printf(
					"temporary accept error: %v; retrying in %s",
					err,
					retryDelay,
				)

				time.Sleep(retryDelay)
				continue
			}

			// Невосстановимую ошибку возвращаем вызывающему коду.
			return fmt.Errorf("accept connection: %w", err)
		}

		// После успешного подключения сбрасываем задержку.
		retryDelay = 0

		// Ждём свободного места, не отклоняя принятое соединение.
		slots <- struct{}{}

		go s.serveClient(conn, slots)
	}
}

// serveClient освобождает место после завершения обработчика.
// Само соединение закрывает существующая функция ServeConn.
// serveClient обрабатывает клиента на выделенном потоке ОС.
func (s *Server) serveClient(conn net.Conn, slots chan struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Освобождаем место после завершения обработки.
	defer func() {
		<-slots
	}()

	// Существующий обработчик принимает файл,
	// отправляет результат и закрывает соединение.
	ServeConn(conn)
}

// rejectClient закрывает соединение при достижении лимита.
func (s *Server) rejectClient(conn net.Conn) {
	log.Printf("client %s: rejected, client limit reached (%d)", conn.RemoteAddr(), s.maxClients)

	if err := conn.Close(); err != nil {
		log.Printf("close rejected connection: %v", err)
	}
}
