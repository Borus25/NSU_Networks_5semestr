package model

type LocationInfo struct {
	Location Location

	Weather      Weather
	WeatherError string

	Places      []PlaceDescription
	PlacesError string
}
