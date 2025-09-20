package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// get hostname, port, subpath from url
func ParseURL(url string) (isHttps bool, hostname string, port int, subpath string, err error) {
	port = 0
	isHttps = strings.HasPrefix(url, "https://")

	protocolEnd := strings.Index(url, "://")
	if protocolEnd == -1 {
		err = fmt.Errorf("invalid url: %s", url)
		return
	}
	hostnameStart := protocolEnd + 3

	hostLeft := url[hostnameStart:]
	portStart := strings.Index(hostLeft, ":")
	subpathStart := strings.Index(hostLeft, "/")

	if portStart == -1 {
		if isHttps {
			port = 443
		} else {
			port = 80
		}
	} else {
		if subpathStart == -1 {
			portStr := hostLeft[portStart+1:]
			port, err = strconv.Atoi(portStr)
		} else {
			portStr := hostLeft[portStart+1:subpathStart]
			port, err = strconv.Atoi(portStr)
		}
		if err != nil {
			err = fmt.Errorf("invalid port in url: %s", url)
			return
		}
	}

	if subpathStart == -1 {
		subpath = "/"
	} else {
		subpath = hostLeft[subpathStart:]
	}

	if portStart == -1 && subpathStart == -1 {
		hostname = hostLeft
		return
	} else if portStart == -1 {
		hostname = hostLeft[:subpathStart]
		return
	} else if subpathStart == -1 {
		hostname = hostLeft[:portStart]
		return
	} else {
		// both port and subpath exist
		if portStart < subpathStart {
			hostname = hostLeft[:portStart]
			return
		}
		err = fmt.Errorf("invalid url: %s for portStart > subpathStart", url)
	}
	
	return
}