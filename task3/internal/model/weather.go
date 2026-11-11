package model

type Weather struct {
	TemperatureC float64
	FeelsLikeC   float64
	HumidityPct  int
	WindSpeedMPS float64
	Description  string
	PressureHPa  *float64
}
