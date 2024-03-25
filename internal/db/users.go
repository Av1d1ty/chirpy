package db

import (
	"fmt"
)

type User struct {
	Id             int    `json:"id"`
	Email          string `json:"email"`
	HashedPassword string `json:"hashed_password"`
}

// CreateUser creates a new user and saves it to disk
func (db *DB) CreateUser(email, hashedPassword string) (User, error) {
	if _, err := db.GetUserByEmail(email); err == nil {
		return User{}, ErrAlreadyExists
	}
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}
	id := len(dbStructure.Users) + 1
	user := User{
		Id:             id,
		Email:          email,
		HashedPassword: hashedPassword,
	}
	dbStructure.Users[id] = user
	err = db.writeDB(dbStructure)
	if err != nil {
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

func (db *DB) UpdateUser(user User) error {
    dbStructure, err := db.loadDB()
    if err != nil {
        return err
    }
    _, ok := dbStructure.Users[user.Id]
    if !ok {
        return ErrNotExist
    }
    dbStructure.Users[user.Id] = user
    return db.writeDB(dbStructure)
}
