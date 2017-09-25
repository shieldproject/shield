package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	cefVersion    = 0
	deviceVendor  = "CloudFoundry"
	deviceProduct = "BOSH"
	deviceVersion = "1"
	signatureID   = "agent_api"
)

type CommonEventFormat interface {
	ProduceHTTPRequestEventLog(*http.Request, int, string) (string, error)
	ProduceNATSRequestEventLog(string, string, string, string, int, string, string) (string, error)
}

func NewCommonEventFormat() CommonEventFormat {
	return concreteCommonEventFormat{}
}

type concreteCommonEventFormat struct{}

func (cef concreteCommonEventFormat) ProduceHTTPRequestEventLog(request *http.Request, respStatusCode int, respBody string) (string, error) {
	name := request.URL.Path
	severity := 1
	if respStatusCode >= 400 {
		severity = 7
	}

	username, _, _ := request.BasicAuth()

	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	headerString := fmt.Sprintf(`HOST=%s&X_REAL_IP=%s&X_FORWARDED_FOR=%s&X_FORWARDED_PROTO=%s&USER_AGENT=%s`,
		request.Header.Get("HTTP_HOST"), request.Header.Get("HTTP_X_REAL_IP"), request.Header.Get("HTTP_X_FORWARDED_FOR"), request.Header.Get("HTTP_X_FORWARDED_PROTO"), request.Header.Get("HTTP_USER_AGENT"))

	extension := fmt.Sprintf(
		`duser=%s requestMethod=%s src=%s spt=%s shost=%s cs1=%s cs1Label=httpHeaders cs2=basic cs2Label=authType cs3=%v cs3Label=responseStatus `,
		username, request.Method, strings.Split(request.RemoteAddr, ":")[0], strings.Split(request.RemoteAddr, ":")[1], hostname, headerString, respStatusCode)

	if respStatusCode >= 400 {
		var buffer bytes.Buffer

		buffer.WriteString(extension)
		buffer.WriteString(fmt.Sprintf("cs4=%s cs4Label=statusReason", respBody))
		extension = buffer.String()
	}

	return fmt.Sprintf("CEF:%v|%s|%s|%s|%s|%s|%v|%s", cefVersion, deviceVendor, deviceProduct, deviceVersion, signatureID, name, severity, extension), nil
}

func (cef concreteCommonEventFormat) ProduceNATSRequestEventLog(addr string, port string, username string, msgMethod string, severity int, subject string, respBody string) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	extension := fmt.Sprintf(
		`duser=%s src=%s spt=%s shost=%s `,
		username, addr, port, hostname)

	if severity >= 7 {
		var buffer bytes.Buffer

		buffer.WriteString(extension)
		buffer.WriteString(fmt.Sprintf("cs1=%s cs1Label=statusReason", respBody))
		extension = buffer.String()
	}

	return fmt.Sprintf("CEF:%v|%s|%s|%s|%s|%s|%v|%s", cefVersion, deviceVendor, deviceProduct, deviceVersion, signatureID, msgMethod, severity, extension), nil
}
