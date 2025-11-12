package withloopperclient

import "os"

func GetHostName() string {
	podName := os.Getenv("HOSTNAME")
	if podName != "" {
		return podName
	}
	return "local_cg"
}
