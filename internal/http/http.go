package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type URLGetter struct {
	client http.Client
}

func NewURLGetter(client http.Client) *URLGetter {
	return &URLGetter{client}
}

func (g *URLGetter) Get(url string) ([]byte, error) {
	res, err := g.client.Get(url)
	if err != nil {
		return []byte{}, fmt.Errorf("getting url %s. %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("status code error: %d %s. %w", res.StatusCode, res.Status, err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("reading body. %w", err)
	}

	return body, nil
}
