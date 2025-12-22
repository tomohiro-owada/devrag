package sample

type User struct {
	ID    int
	Name  string
	Email string
}

type UserService struct {
	users map[int]*User
}

func NewUserService() *UserService {
	return &UserService{users: make(map[int]*User)}
}

func (s *UserService) GetUser(id int) (*User, bool) {
	user, ok := s.users[id]
	return user, ok
}

func (s *UserService) CreateUser(name, email string) *User {
	id := len(s.users) + 1
	user := &User{ID: id, Name: name, Email: email}
	s.users[id] = user
	return user
}

type Greeter interface {
	Greet(name string) string
}
