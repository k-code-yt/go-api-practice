package config

import "fmt"

func GetDefaultConnString() string {
	return "host=localhost port=5432 user=user password=pass dbname=events sslmode=disable"
}

func GetConnString(dbname string) string {
	return fmt.Sprintf("host=localhost port=5432 user=user password=pass dbname=%s sslmode=disable", dbname)
}
