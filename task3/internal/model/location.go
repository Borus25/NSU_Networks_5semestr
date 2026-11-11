package model

type Coordinates struct {
	Lat float64
	Lon float64
}

type Location struct {
	Name        string
	Country     string
	Region      string
	City        string
	Street      string
	HouseNumber string
	OSMKey      string
	OSMValue    string
	Coordinates Coordinates
}
