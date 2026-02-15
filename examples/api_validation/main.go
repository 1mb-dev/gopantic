package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Real-world API structures
type CreateUserRequest struct {
	Username    string   `json:"username" validate:"required,min=3,max=30,alphanum"`
	Email       string   `json:"email" validate:"required,email"`
	Password    string   `json:"password" validate:"required,min=8"`
	FullName    string   `json:"full_name" validate:"required,min=2,max=100"`
	Age         int      `json:"age" validate:"min=13,max=120"`
	Interests   []string `json:"interests"`
	AcceptTerms bool     `json:"accept_terms" validate:"required"`
}

type UpdateUserRequest struct {
	Email     *string  `json:"email" validate:"email"` // Optional but must be valid if provided
	FullName  *string  `json:"full_name" validate:"min=2,max=100"`
	Age       *int     `json:"age" validate:"min=13,max=120"`
	Interests []string `json:"interests"`
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Age       int       `json:"age"`
	Interests []string  `json:"interests"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Mock database
var users []User
var nextID = 1

func main() {
	fmt.Println("Starting API Validation Example Server")
	fmt.Println("This demonstrates real-world API request validation using gopantic")

	// Setup routes
	http.HandleFunc("/users", handleUsers)
	http.HandleFunc("/users/", handleUserByID)

	// Serve static demo page
	http.HandleFunc("/", handleHome)

	fmt.Println("\nServer running on http://localhost:8080")
	fmt.Println("Try these endpoints:")
	fmt.Println("  POST /users - Create a new user")
	fmt.Println("  GET /users - List all users")
	fmt.Println("  PUT /users/{id} - Update a user")
	fmt.Println("\nExample curl commands:")
	fmt.Println(`  curl -X POST http://localhost:8080/users \`)
	fmt.Println(`    -H "Content-Type: application/json" \`)
	fmt.Println(`    -d '{"username":"johndoe","email":"john@example.com","password":"mypassword123","full_name":"John Doe","age":25,"accept_terms":true}'`)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Gopantic API Validation Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .example { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .success { color: #28a745; }
        .error { color: #dc3545; }
        pre { background: #000; color: #fff; padding: 10px; border-radius: 5px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>Gopantic API Validation Demo</h1>
    
    <h2>Available Endpoints</h2>
    
    <div class="example">
        <h3>Create User (POST /users)</h3>
        <p>Creates a new user with validation</p>
        <pre>curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com", 
    "password": "mypassword123",
    "full_name": "John Doe",
    "age": 25,
    "interests": ["programming", "music"],
    "accept_terms": true
  }'</pre>
    </div>

    <div class="example">
        <h3>List Users (GET /users)</h3>
        <p>Returns all users</p>
        <pre>curl http://localhost:8080/users</pre>
    </div>

    <div class="example">
        <h3>Update User (PUT /users/{id})</h3>
        <p>Updates user with partial validation</p>
        <pre>curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newemail@example.com",
    "age": 26
  }'</pre>
    </div>

    <h2>Validation Features</h2>
    <ul>
        <li><strong>Required fields:</strong> username, email, password, full_name, accept_terms</li>
        <li><strong>String length:</strong> username (3-30), full_name (2-100), password (min 8)</li>
        <li><strong>Email format:</strong> email field must be valid email address</li>
        <li><strong>Alphanumeric:</strong> username must be alphanumeric</li>
        <li><strong>Age range:</strong> 13-120 years old</li>
        <li><strong>Partial updates:</strong> PUT requests validate only provided fields</li>
    </ul>

    <h2>Error Examples</h2>
    <p>Try invalid data to see structured error responses:</p>
    
    <div class="example">
        <h3>Invalid Data Example</h3>
        <pre>curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "ab",
    "email": "invalid-email",
    "password": "short",
    "full_name": "",
    "age": 200
  }'</pre>
        <p>This will return detailed validation errors for each field.</p>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleGetUsers(w, r)
	case "POST":
		handleCreateUser(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
	}
}

func handleUserByID(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	path := r.URL.Path
	idStr := path[len("/users/"):]
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "User ID is required", nil)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "User ID must be a number", nil)
		return
	}

	switch r.Method {
	case "GET":
		handleGetUser(w, r, id)
	case "PUT":
		handleUpdateUser(w, r, id)
	case "DELETE":
		handleDeleteUser(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
	}
}

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, users)
}

func handleGetUser(w http.ResponseWriter, r *http.Request, id int) {
	user := findUser(id)
	if user == nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
		return
	}

	writeSuccess(w, http.StatusOK, user)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Read request body
	var body []byte
	if r.Body != nil {
		defer r.Body.Close()
		buf := make([]byte, 1024)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				body = append(body, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
	}

	// Parse and validate request using gopantic
	createReq, err := model.ParseInto[CreateUserRequest](body)
	if err != nil {
		// Convert gopantic errors to structured API response
		if errorList, ok := err.(model.ErrorList); ok {
			// Serialize validation errors to structured format
			if jsonData, jsonErr := errorList.ToJSON(); jsonErr == nil {
				var errorReport model.StructuredErrorReport
				if parseErr := json.Unmarshal(jsonData, &errorReport); parseErr == nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errorReport)
					return
				}
			}
		}

		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
		return
	}

	// Business logic validation (e.g., username uniqueness)
	if findUserByUsername(createReq.Username) != nil {
		writeError(w, http.StatusConflict, "USERNAME_EXISTS", "Username already exists", map[string]string{
			"field": "username",
			"value": createReq.Username,
		})
		return
	}

	if findUserByEmail(createReq.Email) != nil {
		writeError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already exists", map[string]string{
			"field": "email",
			"value": createReq.Email,
		})
		return
	}

	if !createReq.AcceptTerms {
		writeError(w, http.StatusBadRequest, "TERMS_NOT_ACCEPTED", "Terms and conditions must be accepted", nil)
		return
	}

	// Create new user
	now := time.Now()
	user := User{
		ID:        nextID,
		Username:  createReq.Username,
		Email:     createReq.Email,
		FullName:  createReq.FullName,
		Age:       createReq.Age,
		Interests: createReq.Interests,
		CreatedAt: now,
		UpdatedAt: now,
	}

	users = append(users, user)
	nextID++

	writeSuccess(w, http.StatusCreated, user)
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request, id int) {
	// Find existing user
	user := findUser(id)
	if user == nil {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
		return
	}

	// Read request body
	var body []byte
	if r.Body != nil {
		defer r.Body.Close()
		buf := make([]byte, 1024)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				body = append(body, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
	}

	// Parse and validate update request using gopantic
	updateReq, err := model.ParseInto[UpdateUserRequest](body)
	if err != nil {
		// Convert gopantic errors to structured API response
		if errorList, ok := err.(model.ErrorList); ok {
			if jsonData, jsonErr := errorList.ToJSON(); jsonErr == nil {
				var errorReport model.StructuredErrorReport
				if parseErr := json.Unmarshal(jsonData, &errorReport); parseErr == nil {
					writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errorReport)
					return
				}
			}
		}

		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
		return
	}

	// Apply updates only to provided fields
	updated := false

	if updateReq.Email != nil {
		if existingUser := findUserByEmail(*updateReq.Email); existingUser != nil && existingUser.ID != id {
			writeError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already exists", map[string]string{
				"field": "email",
				"value": *updateReq.Email,
			})
			return
		}
		user.Email = *updateReq.Email
		updated = true
	}

	if updateReq.FullName != nil {
		user.FullName = *updateReq.FullName
		updated = true
	}

	if updateReq.Age != nil {
		user.Age = *updateReq.Age
		updated = true
	}

	if updateReq.Interests != nil {
		user.Interests = updateReq.Interests
		updated = true
	}

	if updated {
		user.UpdatedAt = time.Now()

		// Update in "database"
		for i, u := range users {
			if u.ID == id {
				users[i] = *user
				break
			}
		}
	}

	writeSuccess(w, http.StatusOK, user)
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request, id int) {
	// Find user index
	userIndex := -1
	for i, u := range users {
		if u.ID == id {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", nil)
		return
	}

	// Remove user
	users = append(users[:userIndex], users[userIndex+1:]...)

	writeSuccess(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// Helper functions

func findUser(id int) *User {
	for _, user := range users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}

func findUserByUsername(username string) *User {
	for _, user := range users {
		if user.Username == username {
			return &user
		}
	}
	return nil
}

func findUserByEmail(email string) *User {
	for _, user := range users {
		if user.Email == email {
			return &user
		}
	}
	return nil
}

func writeSuccess(w http.ResponseWriter, statusCode int, data interface{}) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: generateRequestID(),
	}

	writeJSON(w, statusCode, response)
}

func writeError(w http.ResponseWriter, statusCode int, code, message string, details interface{}) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		RequestID: generateRequestID(),
	}

	writeJSON(w, statusCode, response)
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// Demonstration function
func init() {
	// Add some sample data
	now := time.Now()
	users = []User{
		{
			ID:        nextID,
			Username:  "demo",
			Email:     "demo@example.com",
			FullName:  "Demo User",
			Age:       25,
			Interests: []string{"programming", "gopantic"},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	nextID++
}
