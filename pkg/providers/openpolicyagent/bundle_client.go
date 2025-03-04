package openpolicyagent

import (
	"bytes"
	"fmt"
	"github.com/hexa-org/policy-orchestrator/pkg/compressionsupport"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
}

type BundleClient struct {
	HttpClient HTTPClient
}

func (b *BundleClient) GetExpressionFromBundle(bundleUrl string, path string) ([]byte, error) {
	get, getErr := b.HttpClient.Get(bundleUrl)
	if getErr != nil {
		return nil, getErr
	}

	all, readErr := io.ReadAll(get.Body)
	if readErr != nil {
		return nil, readErr
	}

	gz, gzipErr := compressionsupport.UnGzip(bytes.NewReader(all))
	if gzipErr != nil {
		return nil, gzipErr
	}

	tarErr := compressionsupport.UnTarToPath(bytes.NewReader(gz), path)
	if tarErr != nil {
		return nil, tarErr
	}
	return os.ReadFile(filepath.Join(path, "/bundle/policy.rego"))
}

// todo - ignoring errors for the moment while spiking

func (b *BundleClient) PostBundle(bundleUrl string, bundle []byte) error {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	formFile, _ := writer.CreateFormFile("bundle", "bundle.tar.gz")
	_, _ = formFile.Write(bundle)
	_ = writer.Close()
	parse, _ := url.Parse(bundleUrl)
	contentType := writer.FormDataContentType()
	_, err := b.HttpClient.Post(fmt.Sprintf("http://%s/bundles", parse.Host), contentType, buf)
	return err
}
