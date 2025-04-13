package oauth

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestGenPassword_CompareHashAndPassword(t *testing.T) {
	assert := assert.New(t)

	// password := "mLA4Kuax3EtxKLk8RmybZJ7eVWlHefh58id1zdrpyRKXvkGL"
	password := uuid.NewString()

	// when
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost) // don't use much more than 10 since: https://stackoverflow.com/questions/69567892/bcrypt-takes-a-lot-of-time-in-go
	assert.Nil(err)
	hashed := string(bytes)
	fmt.Printf("password: %s, password hash: %s", password, hashed)
	
	// then
	err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	assert.Nil(err)

	// not possible to check, because it changes every time it's generated:
	// assert.Equal("$2a$14$fKhsh8SkMPTOVytMf5CeOOrvRl6A7w0HUM5Vrrj1YW5Xxs9V2RzRW", hashed)
}



