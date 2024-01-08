package wheretopark

import (
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
)

var DefaultRestyClient = resty.New()

func init() {
	DefaultRestyClient.GetClient().Timeout = 10 * time.Second
}

func GetString(url *url.URL, headers map[string]string) (string, error) {
	resp, err := DefaultRestyClient.R().Get(url.String())
	if err != nil {
		return "", err
	}
	return resp.String(), nil
}

func Get[T any](url *url.URL, headers map[string]string) (*T, error) {
	resp, err := DefaultRestyClient.R().SetResult(new(T)).Get(url.String())
	if err != nil {
		return nil, err
	}
	response := resp.Result().(*T)
	return response, nil
}
