package scrap

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/apex/log"

	"github.com/dgraph-io/badger/v2"
)

type Retriever interface {
	retrieve(url string) ([]byte, error)
}

type retriever struct{}

func (r retriever) retrieve(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return []byte{}, fmt.Errorf("getting url %s. %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return []byte{}, fmt.Errorf("status code error: %d %s. %w", res.StatusCode, res.Status, err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("reading body. %w", err)
	}

	return body, nil
}

type scraper struct {
	logger log.Interface
	db     *badger.DB
	Retriever
}

type Product struct {
	id            string
	name          string
	price         float64
	originalPrice float64
	imageURL      string
	storeURL      string
}

type jsonProducts struct {
	Product []struct {
		PartNumber        string `json:"partNumber"`
		ProductDetailsURL string `json:"productDetailsUrl"`
		Price             struct {
			SeoPrice              float64 `json:"seoPrice"`
			OriginalProductAmount float64 `json:"originalProductAmount"`
		} `json:"price"`
		Image struct {
			SrcSet struct {
				Src string `json:"src"`
			} `json:"srcSet"`
		} `json:"image"`
	} `json:"tiles"`
}

func NewScraper(logger log.Interface, db *badger.DB) *scraper {
	return &scraper{logger, db, &retriever{}}
}

func (s *scraper) Scrap(urls []string) []Product {
	m := sync.RWMutex{}

	wg := &sync.WaitGroup{}
	wg.Add(len(urls))

	var products []Product

	for _, url := range urls {
		go func(url string) {
			defer wg.Done()

			ps, err := s.scrapSingle(url)
			if err != nil {
				s.logger.WithError(err).Error("scraping url")
			}

			m.Lock()
			defer m.Unlock()

			products = append(products, ps...)
		}(url)
	}

	wg.Wait()

	return products
}

func (s *scraper) scrapSingle(url string) ([]Product, error) {
	body, err := s.retrieve(url)
	if err != nil {
		return []Product{}, fmt.Errorf("retrieving url. %w", err)
	}

	strippedBody := strings.ReplaceAll(strings.ReplaceAll(string(body), " ", ""), "\n", "")

	rgx := regexp.MustCompile(`<script>window.REFURB_GRID_BOOTSTRAP=(.+)};</script>`)
	matches := rgx.FindStringSubmatch(strippedBody)

	if matches == nil || len(matches) < 2 {
		return []Product{}, fmt.Errorf("cannot match regex")
	}

	// first match is the whole <script>...</script>, second one is just the json
	var jsonProducts jsonProducts
	if err := json.Unmarshal([]byte(matches[1]+"}"), &jsonProducts); err != nil {
		return []Product{}, fmt.Errorf("unmarshalling json. %w", err)
	}

	products := make([]Product, len(jsonProducts.Product))

	for i, p := range jsonProducts.Product {
		storeURL := strings.Split(p.ProductDetailsURL, "?")[0]
		splitStoreURL := strings.Split(storeURL, "/")

		product := Product{
			p.PartNumber,
			strings.ReplaceAll(strings.TrimPrefix(splitStoreURL[len(splitStoreURL)-1], "Refurbished-"), "-", " "),
			p.Price.SeoPrice,
			p.Price.OriginalProductAmount,
			strings.Split(p.Image.SrcSet.Src, "?")[0],
			storeURL,
		}
		products[i] = product

		s.logger.WithField("id", product.id).WithField("name", product.name).Info("product found")
	}

	return products, nil
}
