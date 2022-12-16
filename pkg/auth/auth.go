package auth

import (
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

var serverSecret []byte

func GetSecret() []byte {
	if serverSecret == nil {
		serverSecret = []byte(os.Getenv("SECRET"))
	}
	return serverSecret
}

func GetSignedToken(username string, userId int) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"id":       userId,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString(GetSecret())
	if err != nil {
		return "", err
	}
	return t, nil
}

func GetUserId(db *sql.DB, username string, password string) (int, error) {
	query := "SELECT rowId, secret FROM users WHERE userName = ?"
	rows, err := db.Query(query, username)
	if err != nil {
		return -1, err
	}

	var userId int
	var secret string
	if !rows.Next() {
		return -1, errors.New("no matching users")
	}
	if err = rows.Scan(&userId, &secret); err != nil {
		return -1, errors.New("no matching users")
	}
	if err = bcrypt.CompareHashAndPassword([]byte(secret), []byte(password)); err != nil {
		return -1, errors.New("bad password")
	}
	return userId, nil
}
