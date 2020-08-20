package scrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/apex/log"

	"github.com/dgraph-io/badger/v2"
)

type URLGetter interface {
	Get(url string) ([]byte, error)
}

type Scraper struct {
	logger log.Interface
	db     *badger.DB
	getter URLGetter
}

type Product struct {
	id            string
	name          string
	price         float64
	originalPrice float64
	imageURL      string
	storeURL      string
	locale        string
	category      string
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

var ErrCannotMatchRegex = errors.New("cannot match regex")

func NewScraper(logger log.Interface, db *badger.DB, getter URLGetter) *Scraper {
	return &Scraper{logger, db, getter}
}

func (s *Scraper) Scrap(locales []string, categories []string) []Product {
	m := sync.RWMutex{}

	wg := &sync.WaitGroup{}
	wg.Add(len(locales) * len(categories))

	var products []Product

	for _, locale := range locales {
		for _, category := range categories {
			go func(locale, category string) {
				defer wg.Done()

				ps, err := s.scrapSingle(locale, category)
				if err != nil {
					s.logger.WithError(err).Error("scraping url")
				}

				m.Lock()
				defer m.Unlock()

				products = append(products, ps...)
			}(locale, category)
		}
	}

	wg.Wait()

	return products
}

func (s *Scraper) scrapSingle(locale, category string) ([]Product, error) {
	url := fmt.Sprintf("https://www.apple.com/%s/shop/refurbished/%s", locale, category)
	body, err := s.getter.Get(url)

	if err != nil {
		return []Product{}, fmt.Errorf("retrieving url '%s'. %w", url, err)
	}

	strippedBody := strings.ReplaceAll(strings.ReplaceAll(string(body), " ", ""), "\n", "")

	rgx := regexp.MustCompile(`<script>window.REFURB_GRID_BOOTSTRAP=(.+)};</script>`)
	matches := rgx.FindStringSubmatch(strippedBody)

	if matches == nil || len(matches) < 2 {
		return []Product{}, ErrCannotMatchRegex
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
			locale,
			category,
		}
		products[i] = product

		s.logger.WithField("id", product.id).WithField("locale", locale).WithField("name", product.name).Info("product found")
	}

	return products, nil
}
