package config

func GetDefaultConnString() string {
	return "host=localhost port=5432 user=user password=pass dbname=events sslmode=disable"
}
