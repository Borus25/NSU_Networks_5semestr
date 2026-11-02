package model

import "image"

type LocationPreview struct {
	PageID       int64
	ArticleTitle string
	Image        image.Image
	ImageError   string
}
