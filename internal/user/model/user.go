package model

type User struct {
	ID       string `db:"id"`
	Login    string `db:"login"`
	Password []byte `db:"password"`
}
