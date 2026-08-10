package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAccrualStatus_GetOrderStatus Проверка конвертации статуса начисления в статус заказа
func TestAccrualStatus_GetOrderStatus(t *testing.T) {
	tests := []struct {
		name          string
		accrualStatus AccrualStatus
		orderStatus   OrderStatus
	}{
		{
			name:          "positive - Если начисление в статусе 'зарегистрировано', значит заказ нужно перевести в статус 'в процессе'",
			accrualStatus: AccrualStatusRegistered,
			orderStatus:   OrderStatusProcessing,
		},
		{
			name:          "positive - Если начисление в статусе 'в процессе', значит заказ нужно перевести в статус 'в процессе'",
			accrualStatus: AccrualStatusProcessing,
			orderStatus:   OrderStatusProcessing,
		},
		{
			name:          "positive - Если начисление в статусе 'обработан', значит заказ нужно перевести в статус 'обработан'",
			accrualStatus: AccrualStatusProcessed,
			orderStatus:   OrderStatusProcessed,
		},
		{
			name:          "positive - Если начисление в статусе 'ошибка', значит заказ нужно перевести в статус 'ошибка'",
			accrualStatus: AccrualStatusInvalid,
			orderStatus:   OrderStatusInvalid,
		},
		{
			name:          "negative - Если начисление в непонятном статусе, значит заказ нужно перевести в статус 'ошибка'",
			accrualStatus: AccrualStatus("test"),
			orderStatus:   OrderStatusInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			orderStatus := test.accrualStatus.GetOrderStatus()

			assert.Equal(t, test.orderStatus, orderStatus, "При статусе начисления %s ожидался статус заказа %s, но получили %s", test.accrualStatus, test.orderStatus, orderStatus)
		})
	}
}

// TestAccrualStatus_IsFinal Проверка является ли статус конечным
func TestAccrualStatus_IsFinal(t *testing.T) {
	tests := []struct {
		name          string
		accrualStatus AccrualStatus
		isFinal       bool
	}{
		{
			name:          "positive - статус 'Зарегистрирован' не является окончательным",
			accrualStatus: AccrualStatusRegistered,
			isFinal:       false,
		},
		{
			name:          "positive - статус 'В процессе' не является окончательным",
			accrualStatus: AccrualStatusProcessing,
			isFinal:       false,
		},
		{
			name:          "positive - статус 'Обработан' является окончательным",
			accrualStatus: AccrualStatusProcessed,
			isFinal:       true,
		},
		{
			name:          "positive - статус 'Ошибка' является окончательным",
			accrualStatus: AccrualStatusInvalid,
			isFinal:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isFinal := test.accrualStatus.IsFinal()

			if test.isFinal {
				assert.True(t, isFinal, "Статус %s должен быть конечным, но метод выдал false", test.accrualStatus)
			} else {
				assert.False(t, isFinal, "Статус %s не должен быть конечным, но метод выдал true", test.accrualStatus)
			}

		})
	}
}
