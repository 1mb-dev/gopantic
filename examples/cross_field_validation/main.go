package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// UserRegistration demonstrates comprehensive cross-field validation
type UserRegistration struct {
	Username        string `json:"username" validate:"required,min=3,max=20,alphanum"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,password_match"`
	FirstName       string `json:"first_name" validate:"required,min=2,alpha"`
	LastName        string `json:"last_name" validate:"required,min=2,alpha"`
	FullName        string `json:"full_name" validate:"full_name_match"`
}

// AccountSettings demonstrates cross-field validation for account configuration
type AccountSettings struct {
	Email             string `json:"email" validate:"required,email"`
	NotificationEmail string `json:"notification_email" validate:"email,email_different"`
	CurrentPassword   string `json:"current_password" validate:"required,min=8"`
	NewPassword       string `json:"new_password,omitempty" validate:"min=8,password_different"`
	ConfirmPassword   string `json:"confirm_password,omitempty" validate:"new_password_match"`
}

// PriceRange demonstrates numeric cross-field validation
type PriceRange struct {
	MinPrice float64 `json:"min_price" validate:"required,min=0"`
	MaxPrice float64 `json:"max_price" validate:"required,min=0,max_greater_than_min"`
}

func init() {
	// Register password confirmation validator
	model.RegisterGlobalCrossFieldFunc("password_match", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
		confirmPassword, ok := fieldValue.(string)
		if !ok {
			return model.NewValidationError(fieldName, fieldValue, "password_match", "confirm password must be a string")
		}

		// Get the password field
		passwordField := structValue.FieldByName("Password")
		if !passwordField.IsValid() {
			return model.NewValidationError(fieldName, fieldValue, "password_match", "password field not found")
		}

		password := passwordField.String()
		if confirmPassword != password {
			return model.NewValidationError(fieldName, fieldValue, "password_match", "passwords do not match")
		}

		return nil
	})

	// Register full name match validator
	model.RegisterGlobalCrossFieldFunc("full_name_match", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
		fullName, ok := fieldValue.(string)
		if !ok || fullName == "" {
			// Allow empty full name - it's optional
			return nil
		}

		firstNameField := structValue.FieldByName("FirstName")
		lastNameField := structValue.FieldByName("LastName")

		if !firstNameField.IsValid() || !lastNameField.IsValid() {
			return model.NewValidationError(fieldName, fieldValue, "full_name_match", "first name or last name field not found")
		}

		firstName := firstNameField.String()
		lastName := lastNameField.String()
		expectedFullName := firstName + " " + lastName

		if fullName != expectedFullName {
			return model.NewValidationError(fieldName, fieldValue, "full_name_match",
				fmt.Sprintf("full name must match first and last name: expected %q", expectedFullName))
		}

		return nil
	})

	// Register email different validator
	model.RegisterGlobalCrossFieldFunc("email_different", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
		notificationEmail, ok := fieldValue.(string)
		if !ok || notificationEmail == "" {
			// Allow empty notification email
			return nil
		}

		emailField := structValue.FieldByName("Email")
		if !emailField.IsValid() {
			return model.NewValidationError(fieldName, fieldValue, "email_different", "email field not found")
		}

		email := emailField.String()
		if strings.ToLower(notificationEmail) == strings.ToLower(email) {
			return model.NewValidationError(fieldName, fieldValue, "email_different", "notification email must be different from main email")
		}

		return nil
	})

	// Register password different validator
	model.RegisterGlobalCrossFieldFunc("password_different", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
		newPassword, ok := fieldValue.(string)
		if !ok || newPassword == "" {
			// Allow empty new password
			return nil
		}

		currentPasswordField := structValue.FieldByName("CurrentPassword")
		if !currentPasswordField.IsValid() {
			return model.NewValidationError(fieldName, fieldValue, "password_different", "current password field not found")
		}

		currentPassword := currentPasswordField.String()
		if newPassword == currentPassword {
			return model.NewValidationError(fieldName, fieldValue, "password_different", "new password must be different from current password")
		}

		return nil
	})

	// Register new password match validator
	model.RegisterGlobalCrossFieldFunc("new_password_match", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
		confirmPassword, ok := fieldValue.(string)
		if !ok {
			return model.NewValidationError(fieldName, fieldValue, "new_password_match", "confirm password must be a string")
		}

		newPasswordField := structValue.FieldByName("NewPassword")
		if !newPasswordField.IsValid() {
			return model.NewValidationError(fieldName, fieldValue, "new_password_match", "new password field not found")
		}

		newPassword := newPasswordField.String()

		// Only validate if new password is provided
		if newPassword == "" && confirmPassword == "" {
			return nil
		}

		if confirmPassword != newPassword {
			return model.NewValidationError(fieldName, fieldValue, "new_password_match", "password confirmation does not match new password")
		}

		return nil
	})

	// Register numeric comparison validator
	model.RegisterGlobalCrossFieldFunc("max_greater_than_min", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
		maxPrice, ok := fieldValue.(float64)
		if !ok {
			return model.NewValidationError(fieldName, fieldValue, "max_greater_than_min", "max price must be a number")
		}

		minPriceField := structValue.FieldByName("MinPrice")
		if !minPriceField.IsValid() {
			return model.NewValidationError(fieldName, fieldValue, "max_greater_than_min", "min price field not found")
		}

		minPrice := minPriceField.Float()
		if maxPrice <= minPrice {
			return model.NewValidationError(fieldName, fieldValue, "max_greater_than_min",
				fmt.Sprintf("max price (%.2f) must be greater than min price (%.2f)", maxPrice, minPrice))
		}

		return nil
	})
}

func main() {
	fmt.Println("gopantic - Cross-Field Validation Examples")
	fmt.Println("=============================================")

	// Example 1: Valid user registration with cross-field validation
	fmt.Println("\n1. Valid User Registration with Password Confirmation:")
	validUserJSON := `{
		"username": "johndoe123",
		"email": "john.doe@example.com",
		"password": "SecurePass123",
		"confirm_password": "SecurePass123",
		"first_name": "John",
		"last_name": "Doe",
		"full_name": "John Doe"
	}`

	user, err := model.ParseInto[UserRegistration]([]byte(validUserJSON))
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	} else {
		fmt.Printf("User registered successfully: %s (%s)\n", user.Username, user.Email)
		fmt.Printf("   Full Name: %s\n", user.FullName)
		fmt.Printf("   Password confirmation: Matched\n")
	}

	// Example 2: Invalid user registration - passwords don't match
	fmt.Println("\n2. Invalid User Registration (Password Mismatch):")
	invalidPasswordJSON := `{
		"username": "johndoe123",
		"email": "john.doe@example.com",
		"password": "SecurePass123",
		"confirm_password": "DifferentPassword",
		"first_name": "John",
		"last_name": "Doe",
		"full_name": "John Doe"
	}`

	_, err = model.ParseInto[UserRegistration]([]byte(invalidPasswordJSON))
	if err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}

	// Example 3: Invalid user registration - full name doesn't match
	fmt.Println("\n3. Invalid User Registration (Full Name Mismatch):")
	invalidFullNameJSON := `{
		"username": "johndoe123",
		"email": "john.doe@example.com",
		"password": "SecurePass123",
		"confirm_password": "SecurePass123",
		"first_name": "John",
		"last_name": "Doe",
		"full_name": "Jane Smith"
	}`

	_, err = model.ParseInto[UserRegistration]([]byte(invalidFullNameJSON))
	if err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}

	// Example 4: Valid account settings update
	fmt.Println("\n4. Valid Account Settings Update:")
	validSettingsJSON := `{
		"email": "john@example.com",
		"notification_email": "notifications@example.com",
		"current_password": "OldPassword123",
		"new_password": "NewSecurePass456",
		"confirm_password": "NewSecurePass456"
	}`

	settings, err := model.ParseInto[AccountSettings]([]byte(validSettingsJSON))
	if err != nil {
		fmt.Printf("Settings validation failed: %v\n", err)
	} else {
		fmt.Printf("Account settings updated successfully\n")
		fmt.Printf("   Email: %s\n", settings.Email)
		fmt.Printf("   Notification Email: %s Different\n", settings.NotificationEmail)
		fmt.Printf("   Password: Changed and confirmed\n")
	}

	// Example 5: Invalid account settings - same emails
	fmt.Println("\n5. Invalid Account Settings (Same Email Addresses):")
	sameEmailJSON := `{
		"email": "john@example.com",
		"notification_email": "john@example.com",
		"current_password": "OldPassword123",
		"new_password": "NewSecurePass456",
		"confirm_password": "NewSecurePass456"
	}`

	_, err = model.ParseInto[AccountSettings]([]byte(sameEmailJSON))
	if err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}

	// Example 6: Invalid account settings - same password
	fmt.Println("\n6. Invalid Account Settings (Same Password):")
	samePasswordJSON := `{
		"email": "john@example.com",
		"notification_email": "notifications@example.com",
		"current_password": "SamePassword123",
		"new_password": "SamePassword123",
		"confirm_password": "SamePassword123"
	}`

	_, err = model.ParseInto[AccountSettings]([]byte(samePasswordJSON))
	if err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}

	// Example 7: Valid price range
	fmt.Println("\n7. Valid Price Range:")
	validPriceJSON := `{
		"min_price": 10.50,
		"max_price": 99.99
	}`

	priceRange, err := model.ParseInto[PriceRange]([]byte(validPriceJSON))
	if err != nil {
		fmt.Printf("Price validation failed: %v\n", err)
	} else {
		fmt.Printf("Price range valid: $%.2f - $%.2f\n", priceRange.MinPrice, priceRange.MaxPrice)
	}

	// Example 8: Invalid price range - max less than min
	fmt.Println("\n8. Invalid Price Range (Max < Min):")
	invalidPriceJSON := `{
		"min_price": 50.00,
		"max_price": 25.00
	}`

	_, err = model.ParseInto[PriceRange]([]byte(invalidPriceJSON))
	if err != nil {
		fmt.Printf("Expected validation error: %v\n", err)
	}

	// Example 9: Demonstrating optional cross-field validation
	fmt.Println("\n9. Optional Cross-Field Validation:")

	// Full name is optional - should pass without full_name field
	optionalJSON := `{
		"username": "janedoe",
		"email": "jane@example.com",
		"password": "JanePass123",
		"confirm_password": "JanePass123",
		"first_name": "Jane",
		"last_name": "Smith"
	}`

	optionalUser, err := model.ParseInto[UserRegistration]([]byte(optionalJSON))
	if err != nil {
		fmt.Printf("Optional validation failed: %v\n", err)
	} else {
		fmt.Printf("User registered without full name: %s\n", optionalUser.Username)
		fmt.Printf("   Full Name: %q (empty, as expected)\n", optionalUser.FullName)
	}

	fmt.Println("\nAll cross-field validation examples completed!")
	fmt.Println("gopantic successfully demonstrated advanced cross-field validation capabilities!")
}
