package main

import (
	"fmt"
	"log"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// User represents a user with optional profile information
type User struct {
	ID          int        `json:"id" validate:"required,min=1"`
	Username    string     `json:"username" validate:"required,min=3,max=20,alphanum"`
	Email       string     `json:"email" validate:"required,email"`
	FullName    *string    `json:"full_name"`
	Age         *int       `json:"age" validate:"min=13,max=120"`
	Bio         *string    `json:"bio" validate:"max=500"`
	IsActive    *bool      `json:"is_active"`
	Height      *float64   `json:"height" validate:"min=0.1,max=3.0"`
	JoinedAt    time.Time  `json:"joined_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// Profile represents extended user profile information
type Profile struct {
	UserID      int        `json:"user_id" validate:"required,min=1"`
	DisplayName *string    `json:"display_name" validate:"min=2,max=50"`
	Avatar      *string    `json:"avatar"`
	Website     *string    `json:"website"`
	Location    *string    `json:"location" validate:"max=100"`
	Skills      []string   `json:"skills"`
	IsPublic    *bool      `json:"is_public"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

func main() {
	fmt.Println("Gopantic Pointer Types Example")
	fmt.Println("=====================================")
	fmt.Println()

	// Example 1: User with all optional fields provided
	fmt.Println("1. User with all optional fields:")
	userJSON1 := []byte(`{
		"id": "42",
		"username": "johndoe",
		"email": "john@example.com",
		"full_name": "John Doe",
		"age": "28",
		"bio": "Software engineer who loves Go programming",
		"is_active": true,
		"height": "1.8",
		"joined_at": "2023-01-15T10:30:00Z",
		"last_login_at": "2024-01-20T14:45:00Z"
	}`)

	user1, err := model.ParseInto[User](userJSON1)
	if err != nil {
		log.Printf("Error parsing user1: %v", err)
	} else {
		printUser(user1)
	}

	// Example 2: User with minimal required fields only
	fmt.Println("\n2. User with minimal required fields (optional fields will be nil):")
	userJSON2 := []byte(`{
		"id": 99,
		"username": "alice",
		"email": "alice@test.com",
		"joined_at": "2023-06-01T09:00:00Z"
	}`)

	user2, err := model.ParseInto[User](userJSON2)
	if err != nil {
		log.Printf("Error parsing user2: %v", err)
	} else {
		printUser(user2)
	}

	// Example 3: User with explicit null values
	fmt.Println("\n3. User with explicit null values for optional fields:")
	userJSON3 := []byte(`{
		"id": 123,
		"username": "bob",
		"email": "bob@example.org",
		"full_name": null,
		"age": null,
		"bio": null,
		"is_active": null,
		"height": null,
		"joined_at": "2023-12-01T12:00:00Z",
		"last_login_at": null
	}`)

	user3, err := model.ParseInto[User](userJSON3)
	if err != nil {
		log.Printf("Error parsing user3: %v", err)
	} else {
		printUser(user3)
	}

	// Example 4: Profile with optional fields
	fmt.Println("\n4. Profile with optional fields:")
	profileJSON := []byte(`{
		"user_id": 42,
		"display_name": "John the Developer",
		"avatar": "https://example.com/avatar.jpg",
		"website": "https://johndoe.dev",
		"location": "San Francisco, CA",
		"skills": ["Go", "JavaScript", "Docker", "Kubernetes"],
		"is_public": true,
		"created_at": "2023-01-15T10:35:00Z",
		"updated_at": "2023-12-15T16:20:00Z"
	}`)

	profile, err := model.ParseInto[Profile](profileJSON)
	if err != nil {
		log.Printf("Error parsing profile: %v", err)
	} else {
		printProfile(profile)
	}

	// Example 5: Validation error with pointer field
	fmt.Println("\n5. Validation error example (age too low):")
	invalidUserJSON := []byte(`{
		"id": 456,
		"username": "child",
		"email": "child@example.com",
		"age": 10,
		"joined_at": "2024-01-01T00:00:00Z"
	}`)

	_, err = model.ParseInto[User](invalidUserJSON)
	if err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}

	// Example 6: Type coercion with pointers
	fmt.Println("\n6. Type coercion with pointer fields:")
	coercionUserJSON := []byte(`{
		"id": "789",
		"username": "coercion",
		"email": "coercion@test.com",
		"age": "25",
		"is_active": "true",
		"height": "2.1",
		"joined_at": 1640995200,
		"last_login_at": 1672531200
	}`)

	userCoercion, err := model.ParseInto[User](coercionUserJSON)
	if err != nil {
		log.Printf("Error parsing coercion user: %v", err)
	} else {
		fmt.Println("Successfully coerced types:")
		printUser(userCoercion)
	}
}

// Helper function to print user information
func printUser(user User) {
	fmt.Printf("User ID: %d\n", user.ID)
	fmt.Printf("Username: %s\n", user.Username)
	fmt.Printf("Email: %s\n", user.Email)

	if user.FullName != nil {
		fmt.Printf("Full Name: %s\n", *user.FullName)
	} else {
		fmt.Println("Full Name: <not provided>")
	}

	if user.Age != nil {
		fmt.Printf("Age: %d\n", *user.Age)
	} else {
		fmt.Println("Age: <not provided>")
	}

	if user.Bio != nil {
		fmt.Printf("Bio: %s\n", *user.Bio)
	} else {
		fmt.Println("Bio: <not provided>")
	}

	if user.IsActive != nil {
		fmt.Printf("Active: %t\n", *user.IsActive)
	} else {
		fmt.Println("Active: <not provided>")
	}

	if user.Height != nil {
		fmt.Printf("Height: %.1f\n", *user.Height)
	} else {
		fmt.Println("Height: <not provided>")
	}

	fmt.Printf("Joined At: %s\n", user.JoinedAt.Format(time.RFC3339))

	if user.LastLoginAt != nil {
		fmt.Printf("Last Login: %s\n", user.LastLoginAt.Format(time.RFC3339))
	} else {
		fmt.Println("Last Login: <never>")
	}
}

// Helper function to print profile information
func printProfile(profile Profile) {
	fmt.Printf("Profile for User ID: %d\n", profile.UserID)

	if profile.DisplayName != nil {
		fmt.Printf("Display Name: %s\n", *profile.DisplayName)
	} else {
		fmt.Println("Display Name: <not set>")
	}

	if profile.Avatar != nil {
		fmt.Printf("Avatar: %s\n", *profile.Avatar)
	} else {
		fmt.Println("Avatar: <not set>")
	}

	if profile.Website != nil {
		fmt.Printf("Website: %s\n", *profile.Website)
	} else {
		fmt.Println("Website: <not set>")
	}

	if profile.Location != nil {
		fmt.Printf("Location: %s\n", *profile.Location)
	} else {
		fmt.Println("Location: <not set>")
	}

	fmt.Printf("Skills: %v\n", profile.Skills)

	if profile.IsPublic != nil {
		fmt.Printf("Public Profile: %t\n", *profile.IsPublic)
	} else {
		fmt.Println("Public Profile: <not set>")
	}

	fmt.Printf("Created At: %s\n", profile.CreatedAt.Format(time.RFC3339))

	if profile.UpdatedAt != nil {
		fmt.Printf("Updated At: %s\n", profile.UpdatedAt.Format(time.RFC3339))
	} else {
		fmt.Println("Updated At: <never updated>")
	}
}
