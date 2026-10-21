package internal

import "fmt"

func formatSize(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%d Б", size)
	}
	units := [...]string{"Б", "КБ", "МБ", "ГБ"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.2f %s", value, units[unit])
}
