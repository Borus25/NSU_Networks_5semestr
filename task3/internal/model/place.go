package model

import "image"

type PlaceRef struct {
	PageID      int64
	Title       string
	Coordinates Coordinates
	DistanceM   float64
}

type PlaceDescription struct {
	Place   PlaceRef
	Summary string

	ImageURL   string
	Image      image.Image
	ImageError string
}

type PlaceImage struct {
	URL      string
	FileName string
	Width    int
	Height   int
}
