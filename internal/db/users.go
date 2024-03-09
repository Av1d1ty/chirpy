package db

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)


type User struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateUser creates a new user and saves it to disk
func (db *DB) CreateUser(email string, password string) (User, error) {
	if email == "" || password == "" {
		return User{}, fmt.Errorf("Email and password cannot be empty")
	}
    if _, err := db.GetUserByEmail(email); err == nil {
        return User{}, fmt.Errorf("User with this email already exists")
    }
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}
	id := len(dbStructure.Users) + 1
    pwdHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return User{}, err
    }
	user := User{Id: id, Email: email, Password: string(pwdHash)}
	dbStructure.Users[id] = user
	if err := db.writeDB(dbStructure); err != nil {
		return User{}, err
	}
	return user, nil
}

func (db *DB) GetUser(id int) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}
	user, ok := dbStructure.Users[id]
	if !ok {
		return User{}, ErrNotExist
	}
	return user, nil
}

func (db *DB) GetUserByEmail(email string) (User, error) {
	if email == "" {
		return User{}, fmt.Errorf("Email cannot be empty")
	}
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}
    for _, usr := range dbStructure.Users {
        if usr.Email == email {
            return usr, nil
        }
    }
    return User{}, fmt.Errorf("User not found")
}
