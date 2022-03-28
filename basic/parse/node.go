package parse

import "strconv"

const baseName = "node-"
const baseUrl = "localhost:"
const basePort = 1000

func ID2name(ID int) string {
	return baseName + strconv.Itoa(ID)
}

func ID2url(ID int) string {
	port := basePort + ID
	return baseUrl + strconv.Itoa(port)
}
