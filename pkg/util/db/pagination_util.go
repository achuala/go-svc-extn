package db

import (
	"encoding/base64"
	"strconv"
	"strings"
)

func ToPageToken(page, pageSize int) string {
	v := strconv.Itoa(page+1) + "|" + strconv.Itoa(pageSize)
	return base64.RawStdEncoding.EncodeToString([]byte(v))
}

func ParsePageToken(pageToken string) (page, pageSize int) {
	page, pageSize = 1, 10
	if pageToken == "" {
		return page, pageSize
	}
	b, err := base64.RawStdEncoding.DecodeString(pageToken)
	if err != nil {
		return page, pageSize
	}
	v := strings.Split(string(b), "|")
	if len(v) != 2 {
		return page, pageSize
	}
	if page, err = strconv.Atoi(v[0]); err != nil {
		return page, pageSize
	}
	if pageSize, err = strconv.Atoi(v[1]); err != nil {
		return page, pageSize
	}
	return page, pageSize
}
