package ui

import (
	"testing"

	"task3/internal/model"
)

func TestLocationType(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  string
	}{
		{
			name:  "bus stop",
			key:   "highway",
			value: "bus_stop",
			want:  "Автобусная остановка",
		},
		{
			name:  "street",
			key:   "highway",
			value: "residential",
			want:  "Улица",
		},
		{
			name:  "road",
			key:   "highway",
			value: "primary",
			want:  "Автомобильная дорога",
		},
		{
			name:  "building",
			key:   "building",
			value: "yes",
			want:  "Здание",
		},
		{
			name:  "unknown tag preserved",
			key:   "natural",
			value: "peak",
			want:  "natural=peak",
		},
		{
			name: "missing type",
			want: "Не указан",
		},
		{
			name:  "normalized input",
			key:   " HIGHWAY ",
			value: " BUS_STOP ",
			want:  "Автобусная остановка",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location := model.Location{
				OSMKey:   tt.key,
				OSMValue: tt.value,
				// Адресное поле не должно определять тип.
				Street: "Тестовая улица",
			}

			if got := locationType(location); got != tt.want {
				t.Fatalf(
					"locationType() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
