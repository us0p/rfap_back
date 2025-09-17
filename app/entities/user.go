package entities

type User struct {
	Number     string `bson:"number,omitempty"`
	HashedPass string `bson:"hashedpass,omitempty"`
}
