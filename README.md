## Sync expenses from YNAB to splitwise.
// Setup variables from .env
token := os.Getenv("YNAB_TOKEN")
budgetID := os.Getenv("YNAB_BUDGET_ID")
splitwiseAPIKey := os.Getenv("SPLITWISE_KEY")

## How it works

## TODO
Parameterized hardcoded constants.

## Development
go run ./cmd/sync/main.go --dry-run

This should create a cache and prevent you from spamming the APIs.
