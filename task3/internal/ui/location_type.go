package ui

import (
	"strings"

	"task3/internal/model"
)

func locationType(location model.Location) string {
	key := strings.ToLower(strings.TrimSpace(location.OSMKey))
	value := strings.ToLower(strings.TrimSpace(location.OSMValue))

	switch key + "=" + value {
	case "highway=bus_stop":
		return "Автобусная остановка"

	case "public_transport=platform":
		return "Посадочная платформа"

	case "public_transport=stop_position":
		return "Место остановки транспорта"

	case "amenity=bus_station":
		return "Автостанция"

	case "railway=tram_stop":
		return "Трамвайная остановка"

	case "railway=station":
		return "Железнодорожная станция"

	case "railway=halt":
		return "Железнодорожный остановочный пункт"

	case "highway=residential",
		"highway=living_street",
		"highway=pedestrian",
		"highway=unclassified":
		return "Улица"

	case "highway=motorway",
		"highway=trunk",
		"highway=primary",
		"highway=secondary",
		"highway=tertiary",
		"highway=motorway_link",
		"highway=trunk_link",
		"highway=primary_link",
		"highway=secondary_link",
		"highway=tertiary_link",
		"highway=road":
		return "Автомобильная дорога"

	case "highway=service":
		return "Служебный проезд"

	case "highway=footway", "highway=path":
		return "Пешеходная дорожка"

	case "highway=cycleway":
		return "Велосипедная дорожка"

	case "highway=steps":
		return "Лестница"

	case "place=city":
		return "Город"

	case "place=town":
		return "Небольшой город"

	case "place=village":
		return "Село / деревня"

	case "place=hamlet":
		return "Малый населённый пункт"

	case "place=suburb":
		return "Район / пригород"

	case "place=neighbourhood":
		return "Микрорайон"

	case "place=square":
		return "Площадь"

	case "amenity=university":
		return "Университет"

	case "amenity=school":
		return "Школа"

	case "amenity=hospital":
		return "Больница"

	case "amenity=library":
		return "Библиотека"

	case "amenity=cafe":
		return "Кафе"

	case "amenity=restaurant":
		return "Ресторан"

	case "tourism=museum":
		return "Музей"

	case "tourism=hotel":
		return "Гостиница"

	case "tourism=attraction":
		return "Достопримечательность"

	case "leisure=park":
		return "Парк"

	case "leisure=stadium":
		return "Стадион"
	}

	switch key {
	case "building":
		if value == "" || value == "yes" {
			return "Здание"
		}
		return "Здание (" + value + ")"

	case "shop":
		if value == "" || value == "yes" {
			return "Магазин"
		}
		return "Магазин (" + value + ")"
	}

	if key != "" && value != "" {
		return key + "=" + value
	}
	if key != "" {
		return key
	}

	return "Не указан"
}
