package main

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/zheki1/yaprmtrc.git/internal/utils"
)

func sendWithResty(client *resty.Client, url string, body []byte, compress bool, key string) error {
	req := client.R().SetHeader("Content-Type", "application/json")

	if compress {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(body); err != nil {
			return fmt.Errorf("gzip failed: %w", err)
		}
		gz.Close()
		req.SetBody(buf.Bytes())
		req.SetHeader("Content-Encoding", "gzip")
		req.SetHeader("Accept-Encoding", "gzip")
	} else {
		req.SetBody(body)
	}

	if key != "" {
		hash := utils.CalculateHMAC(body, key)
		req.SetHeader("HashSHA256", hash)
	}

	resp, err := req.Post(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("Server returned status %d", resp.StatusCode())
	}

	return nil
}
