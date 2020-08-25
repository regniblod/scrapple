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
	family        string
	color         string
	capacity      string
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
		Filters struct {
			Dimensions struct {
				RefurbClearModel  string `json:"refurbClearModel"`
				DimensionColor    string `json:"dimensionColor"`
				DimensionCapacity string `json:"dimensionCapacity"`
			}
		}
		PartNumber        string `json:"partNumber"`
		Title             string `json:"title"`
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

	rgx := regexp.MustCompile(`window\.REFURB_GRID_BOOTSTRAP = (\n|.+);`)
	matches := rgx.FindStringSubmatch(string(body))

	if matches == nil || len(matches) < 2 {
		return []Product{}, ErrCannotMatchRegex
	}

	// first match is the whole `window.REFURB_GRID_BOOTSTRAP = ...` second one is just the json
	var jsonProducts jsonProducts
	if err := json.Unmarshal([]byte(matches[1]), &jsonProducts); err != nil {
		return []Product{}, fmt.Errorf("unmarshalling json. %w", err)
	}

	products := make([]Product, len(jsonProducts.Product))

	for i, p := range jsonProducts.Product {
		product := Product{
			p.PartNumber,
			p.Filters.Dimensions.RefurbClearModel,
			p.Filters.Dimensions.DimensionColor,
			p.Filters.Dimensions.DimensionCapacity,
			p.Title,
			p.Price.SeoPrice,
			p.Price.OriginalProductAmount,
			strings.Split(p.Image.SrcSet.Src, "?")[0],
			strings.Split(p.ProductDetailsURL, "?")[0],
			locale,
			category,
		}
		products[i] = product

		fmt.Printf("%+v\n", product)
		// s.logger.WithField("id", product.id).WithField("locale", locale).WithField("family", product.Family).WithField("name", product.name).Info("product found")
	}

	return products, nil
}
