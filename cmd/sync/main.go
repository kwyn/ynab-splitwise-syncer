package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aanzolaavila/splitwise.go"
	ynab "github.com/brunomvsouza/ynab.go"
	"github.com/brunomvsouza/ynab.go/api"
	"github.com/brunomvsouza/ynab.go/api/transaction"
	"github.com/joho/godotenv"
)

const (
	NAMESPACE           = "ynab-splitwise"
	LAST_SYNC_CONFIGMAP = "last-sync-date"
	SPLITWISE_GROUP     = 5600408
	KWYN_ID             = 4965744
	SPANG_ID            = 573994
	CATEGORY_GROUP_NAME = "Shared"
	TIMELAYOUT          = "2006-01-02"
)

func main() {
	ctx := context.Background()

	// Define the dry-run flag
	dryRun := flag.Bool("dry-run", false, "Disable actual API calls")
	flag.Parse()
	if *dryRun {
		fmt.Println("Dry run... Printing values")
	}

	// Load .env vairables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Setup variables from .env
	token := os.Getenv("YNAB_TOKEN")
	budgetID := os.Getenv("YNAB_BUDGET_ID")
	splitwiseAPIKey := os.Getenv("SPLITWISE_KEY")

	if token == "" {
		log.Fatal("YNAB_TOKEN is required")
	}

	if splitwiseAPIKey == "" {
		log.Fatal("SPLITWISE_KEY is required")
	}

	lastSyncDate, err := GetLastSyncDate("last-sync-date.txt")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Last Sync Date: %s\n", lastSyncDate.Format(TIMELAYOUT))

	// Initialize the YNAB client
	client := ynab.NewClient(token)

	// Initialize the Slitwise Client
	splitwiseClient := splitwise.Client{
		Token: splitwiseAPIKey,
	}
	categoryMap := getBespokeCategoryMap()
	// Fetch the categories
	categories, err := client.Category().GetCategories(budgetID, nil)
	if err != nil {
		log.Fatalf("Failed to fetch categories: %v", err)
	}
	// Iterate over the categories and fetch transactions for each category in the desired category group
	for _, categoryGroup := range categories.GroupWithCategories {
		fmt.Println(categoryGroup.Name)
		if categoryGroup.Name == CATEGORY_GROUP_NAME {
			for _, category := range categoryGroup.Categories {
				fmt.Printf("Category Name and ID %s: %s\n", category.Name, category.ID)

				f := &transaction.Filter{
					Since: &api.Date{lastSyncDate},
				}
				transactions, err := client.Transaction().GetTransactionsByCategory(budgetID, category.ID, f)
				if err != nil {
					log.Fatalf("Failed to fetch transactions for category %s: %v", category.ID, err)
				}

				for _, tx := range transactions {
					var categoryName, payeeName, memo string

					// Check if the pointers are nil before dereferencing them
					if tx.CategoryName != nil {
						categoryName = *tx.CategoryName
					}
					if tx.PayeeName != nil {
						payeeName = *tx.PayeeName
					}
					if tx.Memo != nil {
						memo = *tx.Memo
					}

					// Skip any non-negative transactions (aka credits)
					if tx.Amount < 0 {
						amount := CentsToDollars(tx.Amount * -1)
						description := fmt.Sprintf("ID: %s\n, Category: %s\n Payee: %s\n Memo: %s\n Amount: %.2f\n", tx.ID, categoryName, payeeName, memo, amount)
						name := *tx.CategoryName
						params := splitwise.CreateExpenseParams{
							"details":     description,
							"date":        tx.Date.Format(TIMELAYOUT),
							"category_id": categoryMap[category.ID]}

						if *dryRun {
							// If dry-run is enabled, just log the information
							log.Printf("Will create expense with name: %s, amount: %f, params: %v\n", name, amount, params)
						} else {
							// If dry-run is not enabled, make the actual API call
							_, err := splitwiseClient.CreateExpenseEqualGroupSplit(ctx, amount, name, SPLITWISE_GROUP, params)
							if err != nil {
								fmt.Fprintf(os.Stderr, "could not create expense: %v", err)
							}
						}
					}

				}
			}
		}

	}
	if !*dryRun {
		err = UpdateLastSyncDate("last-sync-date.txt")
		if err != nil {
			log.Fatal(err)
		}
	}

}

// GetLastSyncDate retrieves the last synchronized date from a file
func GetLastSyncDate(filename string) (time.Time, error) {
	// Check if the file exists
	_, err := os.Stat(filename)

	if os.IsNotExist(err) {
		// If the file does not exist, return an empty string
		return time.Time{}, nil
	} else if err != nil {
		// If there was an error checking the file, return the error
		return time.Time{}, nil
	}

	// Read the file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return time.Time{}, nil
	}

	// Parse time
	t, err := time.Parse(TIMELAYOUT, string(data))
	if err != nil {
		return time.Time{}, nil
	}

	// Return the file contents as a string
	return t, nil
}

// UpdateLastSyncDate updates the last synchronized date in a file
func UpdateLastSyncDate(filename string) error {
	// Get the current date
	currentDate := time.Now().Format("2006-01-02")

	// Write the current date to the file
	err := ioutil.WriteFile(filename, []byte(currentDate), 0644)
	if err != nil {
		return err
	}

	return nil
}

// CentsToDollars converts an amount in cents to dollars.
func CentsToDollars(cents int64) float64 {
	return float64(cents) / 1000.0
}

func getBespokeCategoryMap() map[string]int {
	return map[string]int{
		"60037d9a-1a2e-4960-b067-f9d5d548d8ec": 1,  // Amazon Prime: Utilities
		"4611dd30-b5c8-479b-a59c-b2794dd2d4f0": 28, // Cabin Improvements : Home(Other)
		"381373d1-56d3-4929-bb4c-c19abb41b8e6": 17, // Cabin Maintence: Maintence
		"9b8a7501-2aa8-42e1-93dc-68cc8a6a1e95": 37, // Cabin Trash: Trash
		"139acc44-1191-4c55-9768-b3a859bbf9a6": 12, // Groceries:
		"70954a26-5c5b-4483-8e62-06af2a76d4df": 7,  // Cabin Water: Water
		"bf72c56f-8041-4000-b72d-93ad9fb44931": 4,  // Oakland Mortgage: Mortgage
		"5cc4da89-36e3-4f55-90a7-9853a958feae": 8,  // Apple TV+ : TV
		"2d0eed50-4fbc-4694-b5be-54b340e3c8d0": 9,  // Youtube Premium: TV
		"5c7bd50f-af65-417f-84c8-f035f25c5d62": 8,  // Oakland Internet
		"f1b7b3bb-b8ad-4831-bfd8-92f542c0c937": 6,  // Cabin Propane
		"a5782661-903e-4989-a52b-fcf9d259f50b": 5,  // PG&E
		"93de32c8-f916-44b8-b959-afa6ce0d2abf": 8,  // Garmen Inreach
		"f0422845-46a3-433d-961d-d214592b8e29": 10, // Car Insurance
		"730e72a6-1bfc-453b-8b9c-ba2ebab8ad86": 28, // Oakland Gardening
		"f770a5db-4835-494e-ab45-f1364850613f": 8,  // Cabin Internet
	}
}
