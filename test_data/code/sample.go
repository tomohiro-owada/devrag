package sample

// User represents a user in the system
type User struct {
	ID    int
	Name  string
	Email string
}

// UserService provides user-related operations
type UserService struct {
	users map[int]*User
}

// NewUserService creates a new UserService instance
func NewUserService() *UserService {
	return &UserService{
		users: make(map[int]*User),
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(id int) (*User, bool) {
	user, exists := s.users[id]
	return user, exists
}

// CreateUser creates a new user
func (s *UserService) CreateUser(name, email string) *User {
	id := len(s.users) + 1
	user := &User{
		ID:    id,
		Name:  name,
		Email: email,
	}
	s.users[id] = user
	return user
}

// Greeter interface for greeting functionality
type Greeter interface {
	Greet(name string) string
}

// Hello is a simple greeting function
func Hello(name string) string {
	return "Hello, " + name + "!"
}
