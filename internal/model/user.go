package model

type User struct {
	ID       string `json:"id" bson:"_id,omitempty"`
	Username string `json:"username" bson:"un"`
	Password string `json:"password,omitempty" bson:"pwd,omitempty"`
	Email    string `json:"email" bson:"email"`
}
