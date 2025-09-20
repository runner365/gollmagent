package httpclient

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
)

func HTTPSPost(hostname string, port int, subpath string, data []byte, header http.Header) ([]byte, error) {
	targetURL := fmt.Sprintf("https://%s:%d%s", hostname, port, subpath)

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create request error: %v", err)
	}

	req.Header = header

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request error: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response error: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("server returned error status code: %d, status:%s",
			resp.StatusCode, resp.Status)
	}

	return respBody, nil
}
