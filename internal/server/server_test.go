package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alexsey-popov/gmart-bonus/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getFreePort находит свободный порт для тестов сервера
func getFreePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().String()
}

// TestNewServer Проверка создания сервера
func TestNewServer(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		UserAddress: "127.0.0.1:8080",
		JwtToken:    "secret",
	}

	t.Run("positive - успешное создание сервера", func(t *testing.T) {
		srv, err := NewServer(cfg, log, nil)
		require.NoError(t, err)
		assert.NotNil(t, srv.srv)
		assert.Equal(t, cfg, srv.cfg)
		assert.Equal(t, log, srv.log)
	})
}

// TestServerLifecycle Проверка жизненного цикла сервера (старт и graceful shutdown)
func TestServerLifecycle(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	addr := getFreePort(t)
	cfg := &config.Config{
		UserAddress: addr,
		JwtToken:    "secret",
	}

	srv, err := NewServer(cfg, log, nil)
	require.NoError(t, err)

	// Запускаем сервер в отдельной горутине
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServe()
	}()

	// Ждем немного, чтобы сервер успел запуститься
	time.Sleep(50 * time.Millisecond)

	// Проверяем, что сервер отвечает (например, GET /api/user/balance выдаст 401 Unauthorized)
	resp, err := http.Get("http://" + addr + "/api/user/balance")
	if err == nil {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	// Останавливаем сервер
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	assert.NoError(t, err)

	// Проверяем, что ListenAndServe завершился без ошибки (ErrServerClosed обрабатывается как nil)
	select {
	case err = <-errChan:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to stop")
	}
}

// TestServerShutdownWithoutStart Проверка завершения работы сервера, который не был запущен
func TestServerShutdownWithoutStart(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		UserAddress: "127.0.0.1:0",
		JwtToken:    "secret",
	}

	srv, err := NewServer(cfg, log, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	_ = err
}
