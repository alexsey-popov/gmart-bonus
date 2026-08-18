package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Проверка является ли статус конечным
func TestOrderStatus_IsFinal(t *testing.T) {
	tests := []struct {
		name        string
		orderStatus OrderStatus
		isFinal     bool
	}{
		{
			name:        "positive - статус 'Новый' не является окончательным",
			orderStatus: OrderStatusNew,
			isFinal:     false,
		},
		{
			name:        "positive - статус 'В процессе' не является окончательным",
			orderStatus: OrderStatusProcessing,
			isFinal:     false,
		},
		{
			name:        "positive - статус 'Обработан' является окончательным",
			orderStatus: OrderStatusProcessed,
			isFinal:     true,
		},
		{
			name:        "positive - статус 'Ошибка' является окончательным",
			orderStatus: OrderStatusInvalid,
			isFinal:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isFinal := test.orderStatus.IsFinal()

			if test.isFinal {
				assert.True(t, isFinal, "Статус %s должен быть конечным, но метод выдал false", test.orderStatus)
			} else {
				assert.False(t, isFinal, "Статус %s не должен быть конечным, но метод выдал true", test.orderStatus)
			}

		})
	}
}
