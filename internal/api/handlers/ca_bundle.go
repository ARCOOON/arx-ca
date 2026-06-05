package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
)

type caBundleInput struct {
	RootPEM         string
	IntermediatePEM string
}

func buildConcatenatedCAChain(intermediatePEM, rootPEM string) string {
	intermediate := strings.TrimSpace(intermediatePEM)
	root := strings.TrimSpace(rootPEM)

	chain := intermediate
	if !strings.HasSuffix(chain, "\n") {
		chain += "\n"
	}
	chain += root
	if !strings.HasSuffix(chain, "\n") {
		chain += "\n"
	}
	return chain
}

func buildCABundleZip(input caBundleInput) ([]byte, error) {
	intermediatePEM := strings.TrimSpace(input.IntermediatePEM)
	if intermediatePEM == "" {
		return nil, fmt.Errorf("intermediate certificate is required")
	}
	rootPEM := strings.TrimSpace(input.RootPEM)
	if rootPEM == "" {
		return nil, fmt.Errorf("root certificate is required")
	}

	if !strings.HasSuffix(intermediatePEM, "\n") {
		intermediatePEM += "\n"
	}
	if !strings.HasSuffix(rootPEM, "\n") {
		rootPEM += "\n"
	}

	chain := buildConcatenatedCAChain(intermediatePEM, rootPEM)

	entries := []struct {
		name    string
		content string
	}{
		{name: "root.pem", content: rootPEM},
		{name: "root.crt", content: rootPEM},
		{name: "intermediate.pem", content: intermediatePEM},
		{name: "intermediate.crt", content: intermediatePEM},
		{name: "ca-chain.pem", content: chain},
		{name: "ca-chain.crt", content: chain},
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, entry := range entries {
		file, err := writer.Create(entry.name)
		if err != nil {
			return nil, fmt.Errorf("create zip entry %q: %w", entry.name, err)
		}
		if _, err := file.Write([]byte(entry.content)); err != nil {
			return nil, fmt.Errorf("write zip entry %q: %w", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize zip archive: %w", err)
	}

	return buf.Bytes(), nil
}
