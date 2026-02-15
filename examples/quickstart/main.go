package main

import (
	"fmt"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// User demonstrates basic parsing with validation
type User struct {
	ID    int    `json:"id" validate:"required,min=1"`
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=18,max=120"`
}

// Product demonstrates validation with type coercion
type Product struct {
	SKU   string  `json:"sku" validate:"required,length=8"`
	Price float64 `json:"price" validate:"required,min=0.01"`
	Stock int     `json:"stock" validate:"min=0"`
}

func main() {
	fmt.Println("gopantic - Quickstart Examples")
	fmt.Println("================================")
	fmt.Println()

	// Example 1: Basic parsing with validation
	fmt.Println("1. Basic Parsing and Validation:")
	userJSON := `{
		"id": 42,
		"name": "Alice Johnson",
		"email": "alice@example.com",
		"age": 28
	}`

	user, err := model.ParseInto[User]([]byte(userJSON))
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   User: %s (%s), Age: %d\n", user.Name, user.Email, user.Age)
	}

	// Example 2: Type coercion (strings to numbers)
	fmt.Println("\n2. Automatic Type Coercion:")
	coercionJSON := `{
		"id": "123",
		"name": "Bob Smith",
		"email": "bob@example.com",
		"age": "35"
	}`

	user2, err := model.ParseInto[User]([]byte(coercionJSON))
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   String '123' → int %d\n", user2.ID)
		fmt.Printf("   String '35' → int %d\n", user2.Age)
	}

	// Example 3: Validation errors
	fmt.Println("\n3. Validation Errors:")
	invalidJSON := `{
		"id": 0,
		"name": "X",
		"email": "invalid-email",
		"age": 15
	}`

	_, err = model.ParseInto[User]([]byte(invalidJSON))
	if err != nil {
		fmt.Printf("   Multiple validation errors caught:\n   %v\n", err)
	}

	// Example 4: Product with validation
	fmt.Println("\n4. Product Validation:")
	validProduct := `{
		"sku": "WDG12345",
		"price": "99.99",
		"stock": "50"
	}`

	product, err := model.ParseInto[Product]([]byte(validProduct))
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Product: %s at $%.2f (%d in stock)\n", product.SKU, product.Price, product.Stock)
	}

	// Example 5: Invalid product
	fmt.Println("\n5. Invalid Product:")
	invalidProduct := `{
		"sku": "SHORT",
		"price": "-10.00",
		"stock": "-5"
	}`

	_, err = model.ParseInto[Product]([]byte(invalidProduct))
	if err != nil {
		fmt.Printf("   Validation failed:\n   %v\n", err)
	}

	fmt.Println("\nQuickstart completed!")
	fmt.Println("See other examples for advanced features:")
	fmt.Println("  - yaml/ for YAML parsing")
	fmt.Println("  - cache_demo/ for high-performance caching")
	fmt.Println("  - cross_field_validation/ for custom validators")
}
