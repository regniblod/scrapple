package scrap

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/apex/log"

	"github.com/dgraph-io/badger/v2"
)

//type Retriever interface {
//	retrieve(url string) ([]byte, error)
//}

type Scraper struct {
	logger log.Interface
	db     *badger.DB
	//Retriever
}

type Product struct {
	id            string
	name          string
	price         float64
	originalPrice float64
	imageURL      string
	storeURL      string
}

func NewScraper(logger log.Interface, db *badger.DB) *Scraper {
	return &Scraper{logger, db}
}

func (s *Scraper) Scrap(urls []string) error {
	wg := &sync.WaitGroup{}
	wg.Add(len(urls))

	for _, url := range urls {
		go func(url string) {
			s.scrapSingle(url)
			wg.Done()
		}(url)
	}

	wg.Wait()

	return nil
}

func (s *Scraper) scrapSingle(url string) []Product {
	res, err := http.Get(url)
	if err != nil {
		s.logger.WithError(err).Errorf("getting %s", url)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		s.logger.Errorf("status code error: %d %s", res.StatusCode, res.Status)
		return nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		s.logger.WithError(err).Error("reading http response")
		return nil
	}

	strippedBody := strings.ReplaceAll(strings.ReplaceAll(string(body), " ", ""), "\n", "")

	rgx := regexp.MustCompile(`<script>window.REFURB_GRID_BOOTSTRAP=(.+)};</script>`)
	matches := rgx.FindStringSubmatch(strippedBody)

	if matches == nil || len(matches) < 2 {
		s.logger.Error("cannot match regex")
		return nil
	}

	// first match is the whole <script>...</script>, second one is just the json
	var jsonProducts map[string]interface{}
	err = json.Unmarshal([]byte(matches[1]+"}"), &jsonProducts)
	if err != nil {
		s.logger.WithError(err).Error("unmarshalling json")
		return nil
	}

	var products []Product
	tiles := jsonProducts["tiles"].([]interface{})
	for _, tile := range tiles {
		jsonProduct := tile.(map[string]interface{})
		price := jsonProduct["price"].(map[string]interface{})
		image := jsonProduct["image"].(map[string]interface{})
		storeURL := strings.Split(jsonProduct["productDetailsUrl"].(string), "?")[0]
		splitStoreURL := strings.Split(storeURL, "/")

		product := Product{
			jsonProduct["partNumber"].(string),
			strings.ReplaceAll(strings.TrimPrefix(splitStoreURL[len(splitStoreURL)-1], "Refurbished-"), "-", " "),
			price["seoPrice"].(float64),
			price["originalProductAmount"].(float64),
			strings.Split(image["srcSet"].(map[string]interface{})["src"].(string), "?")[0],
			storeURL,
		}
		products = append(products, product)

		s.logger.WithField("id", product.id).WithField("name", product.name).Info("product found")
	}

	return products
}
