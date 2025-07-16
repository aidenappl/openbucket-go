package logs

import (
	"log"
)

func Println(request string, host string, message ...interface{}) {
	log.Printf("RequestId: %s; HostId: %s; Message: %s", request, host, fmt.Sprint(message...))
}
