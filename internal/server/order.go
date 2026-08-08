// Пакет для реализации движения заказов по статусам.
// Берём заказы в незаконченных статусах и обращаемся по ним в систему расчёта бонусов.
// В зависимости от ответа переводим заказ в статусы PROCESSING/INVALID/PROCESSED
// В случае PROCESSED корректируем бонусы пользователя
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/alexsey-popov/gmart-bonus/internal/model"
	"github.com/alexsey-popov/gmart-bonus/internal/repository"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

const (
	// EmptyBatchPause Время ожидания при получении пустой пачки данных
	EmptyBatchPause = 10 * time.Second

	// CountWorkers Количество обработчиков для startOrderProcessing
	CountWorkers = 10
)

// ErrWithPause ошибка обработки, содержащая количество секунд,
// на которое необходимо приостановить все обработки.
type ErrWithPause struct {
	Seconds time.Duration
	Err     error
}

// Error Реализуем интерфейс error
func (e *ErrWithPause) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("обработка приостановлена на %d сек.: %v", e.Seconds, e.Err)
	}
	return fmt.Sprintf("обработка приостановлена на %d сек.", e.Seconds)
}

// Unwrap Разворачиваем ошибку
func (e *ErrWithPause) Unwrap() error {
	return e.Err
}

// NewErrWithPause Создание новой ошибки с указанием времени
func NewErrWithPause(seconds int) error {
	return &ErrWithPause{
		Seconds: time.Duration(seconds) * time.Second,
		Err:     errors.New("превышен лимит запросов"),
	}
}

// LoopOrderProcessing Циклическая обработка заказов (прерывается по контексту)
func (s Server) LoopOrderProcessing(ctx context.Context) error {
	for {
		s.log.Info("Запуск обработки заказов в системе начисления")

		// Если пришла отмена контекста - не начинаем новую обработку
		if err := ctx.Err(); err != nil {
			return nil
		}

		// Получаем пачку заказов
		orders, err := s.rep.GetUnfinishedOrders(ctx)
		if err != nil {
			return err
		}

		// Если нам пришла пустая пачка данных - значит обрабатывать нечего и вместо того,
		// чтобы бесконечно запрашивать данные у БД, лучше подождать некоторое время
		if len(orders) == 0 {
			if err = s.pauseOrderProcessing(ctx, EmptyBatchPause); err != nil {
				return err
			}
			continue
		}

		// Обрабатываем полученные записи
		if err = s.startOrderProcessing(ctx, s.rep, orders); err != nil {
			// Если произошла PauseError, значит приостанавливаем обработку на указанное время
			var pauseErr *ErrWithPause
			if errors.As(err, &pauseErr) {
				if err2 := s.pauseOrderProcessing(ctx, pauseErr.Seconds); err2 != nil {
					return err2
				}
				continue
			}

			return err
		}

		// После каждой пачки записей отдыхаем 10 сек
		if err = s.pauseOrderProcessing(ctx, EmptyBatchPause); err != nil {
			return err
		}
	}
}

// pauseOrderProcessing Остановка выполнения обработки на заданное время или до отмены контекста
func (s Server) pauseOrderProcessing(ctx context.Context, d time.Duration) error {
	s.log.Info("Пауза в обработке заказов системой начисления",
		slog.String("duration", d.String()),
	)
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// startOrderProcessing Обработка заказов пулом горутин.
func (s Server) startOrderProcessing(ctx context.Context, rep *repository.Repository, records []model.Order) error {
	pg, pCtx := errgroup.WithContext(ctx)
	ordersCh := make(chan model.Order)

	// Раздаём записи по каналу
	pg.Go(func() error {
		defer close(ordersCh)
		for _, order := range records {
			select {
			case <-pCtx.Done():
				return pCtx.Err()
			case ordersCh <- order:
			}
		}
		return nil
	})

	// Запускаем указанное количество обработчиков
	for i := 0; i < CountWorkers; i++ {
		pg.Go(func() error {
			for order := range ordersCh {
				// Каждый обработчик актуализирует данных по заказу
				if _, err := s.ActualizeOrder(pCtx, order); err != nil {
					return err
				}
			}
			return nil
		})
	}

	return pg.Wait()
}

// Actualize Актуализация данных из системы расчёта бонусов
func (s Server) ActualizeOrder(ctx context.Context, order model.Order) (model.Order, error) {
	// Если заказ находится в конечных статусах - просто возвращаем его обратно
	if !slices.Contains(model.UnfinishedOrderStatuses, order.Status) {
		return order, nil
	}

	// Делаем запрос в систему расчёта бонусов

	urlPath, err := url.JoinPath(s.cfg.AccrualAddress, "/api/orders/", order.Number)
	if err != nil {
		return order, fmt.Errorf("ошибка при получении эндпоинта системы начисления: %w", err)
	}

	client := &http.Client{}
	response, err := client.Get(urlPath)
	if err != nil {
		return order, fmt.Errorf("ошибка при отправке запроса в систему расчёта бонусов :%w", err)
	}
	defer response.Body.Close()

	// Далее действия различаются в зависимости от кода ответа
	switch response.StatusCode {

	// Данные пришли успешно, можно получать и обрабатывать ответ
	case http.StatusOK:
		// Парсим данные
		var accrual model.AccrualOrder
		if err = json.NewDecoder(response.Body).Decode(&accrual); err != nil {
			err = fmt.Errorf("Ошибка при декодировании ответа от системы расчёта бонусов : %w", err)

			s.log.Error(err.Error(),
				slog.String("url", urlPath),
			)

			// Ошибку не возвращаем, чтобы не сломать дальнейшую обработку
			return order, nil
		}

		// Получаем статус заказа и обновляем его в базу
		status := accrual.GetOrderStatus()
		s.rep.UpdateOrderStatus(ctx, order, status, accrual.Accrual)

	// Номер заказа не зарегистрирован в системе расчёта бонусов - такие заказы сразу переводим в INVALID
	case http.StatusNoContent:
		s.rep.UpdateOrderStatus(ctx, order, model.OrderStatusInvalid, &decimal.NullDecimal{})

	// Превышен лимит запросов? - нужно остановить обработку на указанное в Retry-After количество секунд
	case http.StatusTooManyRequests:
		// Получаем значение из заголовка Retry-After, по умолчанию считаем 60 сек
		retry, err := strconv.Atoi(response.Header.Get("Retry-After"))
		if err != nil {
			retry = 60
		}

		s.log.Info("Система начисления вернула Retry-After",
			slog.Any("retry", retry),
		)

		return order, NewErrWithPause(retry)

	// Ошибка сервера? - просто логируем. Ошибку не возвращаем для того, чтобы продолжалась обработка
	case http.StatusInternalServerError:
		s.log.Info("Ошибка при выполнении запроса в систему расчёта начислений",
			slog.String("url", urlPath),
		)

	// Нестандартный код ответа - логируем. Ошибку не возвращаем для того, чтобы продолжалась обработка
	default:
		s.log.Error("Систему расчёта начислений вернула нестандартный код ответа",
			slog.String("url", urlPath),
			slog.Int("status_code", response.StatusCode),
		)
	}

	return order, nil
}
