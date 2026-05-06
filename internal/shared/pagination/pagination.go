package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

type Params struct {
	Page    int
	PerPage int
	Offset  int
}

// ExtractPagination parses pagination query params from gin.Context.
func ExtractPagination(c *gin.Context) Params {
	page := getQueryInt(c, "page", DefaultPage)
	perPage := getQueryInt(c, "per_page", DefaultPerPage)

	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	offset := (page - 1) * perPage

	return Params{
		Page:    page,
		PerPage: perPage,
		Offset:  offset,
	}
}

func getQueryInt(c *gin.Context, key string, fallback int) int {
	val := c.Query(key)
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}
