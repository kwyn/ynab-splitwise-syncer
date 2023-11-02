package ynab

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brunomvsouza/ynab.go"
	"github.com/brunomvsouza/ynab.go/api/transaction"
)

const (
	cacheFolder                    = "ynab_cache"
	transactionCacheTemplate       = "transactions_%s_%s.txt"
	groupedCategoriesCacheTemplate = "grouped_categories_%s_%s.txt"
)

type CategoryGroupMap map[string]map[string]string

// CachedClient is a naive cached client that only caches for the day so there should only be one call per day to the the ynab API
type CachedClient struct {
	budgetID string
	client   ynab.ClientServicer
}

func NewCachedClient(token string, budgetID string) *CachedClient {
	return &CachedClient{
		budgetID: budgetID,
		client:   ynab.NewClient(token),
	}
}

func (c *CachedClient) ensureCacheFolderExists() error {
	if _, err := os.Stat(cacheFolder); os.IsNotExist(err) {
		return os.Mkdir(cacheFolder, 0755)
	}
	return nil
}

func (c *CachedClient) formatCacheName(apiCall string) string {
	cacheTemplate := ""
	switch apiCall {
	case "groupedCategories":
		cacheTemplate = groupedCategoriesCacheTemplate
	case "transactions":
		cacheTemplate = transactionCacheTemplate
	}
	return fmt.Sprintf(filepath.Join(cacheFolder, cacheTemplate), c.budgetID, time.Now().Format("2006-01-02"))
}

func (c *CachedClient) saveToCache(apiCall string, data []byte) error {
	if err := c.ensureCacheFolderExists(); err != nil {
		return err
	}
	return ioutil.WriteFile(c.formatCacheName(apiCall), data, 0644)
}

func (c *CachedClient) readFromCache(apiCall string) ([]byte, error) {

	return ioutil.ReadFile(c.formatCacheName(apiCall))
}

func (c *CachedClient) CategoryGroupMap() (map[string]string, error) {

	cachedData, err := c.readFromCache("groupedCategories")
	if err == nil {
		fmt.Println("Reading category group from cache...")
		// Unmarshal the cached data and return
		var categoryGroupMap *map[string]string
		err := json.Unmarshal(cachedData, &categoryGroupMap)
		if err != nil {
			return nil, err
		}
		return *categoryGroupMap, nil
	}

	categories, err := c.client.Category().GetCategories(c.budgetID, nil)
	if err != nil {
		log.Fatalf("Failed to fetch categories: %v", err)
	}

	// Iterate over the category groups
	categoryGroupMap := make(map[string]string)
	for _, group := range categories.GroupWithCategories {
		for _, category := range group.Categories {
			categoryGroupMap[category.ID] = group.Name
		}
	}

	data, err := json.Marshal(categoryGroupMap)
	if err != nil {
		return nil, err
	}

	c.saveToCache("groupedCategories", data)

	return categoryGroupMap, nil
}

func (c *CachedClient) GetTransactions(f *transaction.Filter) ([]*transaction.Transaction, error) {
	// Check if we have a cached result from today
	cachedData, err := c.readFromCache("transactions")
	if err == nil {
		fmt.Println("Reading transactions from cache...")
		// Unmarshal the cached data and return
		var transactions []*transaction.Transaction
		err := json.Unmarshal(cachedData, &transactions)
		if err != nil {
			return nil, err
		}
		return transactions, nil
	}

	fmt.Println("No cache found fetching transactions...")
	// If not, make the API call
	transactions, err := c.client.Transaction().GetTransactions(c.budgetID, f)
	if err != nil {
		return nil, err
	}

	// Save the result to the cache
	data, err := json.Marshal(transactions)
	if err != nil {
		return nil, err
	}
	err = c.saveToCache("transactions", data)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}
