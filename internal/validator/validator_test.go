package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Request struct {
	Name string `validate:"required,min=3,max=255" label:"Логин"`
}

// Проверка функции Validate у структуры Validator
func TestValidator_Validate(t *testing.T) {
	v, err := NewValidator()
	require.NoError(t, err)

	tests := []struct {
		name    string
		req     Request
		wantErr bool
		errMsg  string
	}{
		{
			name: "positive - Корректные данные",
			req: Request{
				Name: "test",
			},
			wantErr: false,
		},
		{
			name: "negative - проверка длины поля",
			req: Request{
				Name: "te",
			},
			wantErr: true,
		},
		{
			name: "negative - проверка обязательности поля",
			req: Request{
				Name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
