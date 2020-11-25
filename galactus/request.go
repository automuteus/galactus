package galactus

type UserModify struct {
	UserID string `json:"userID"`
	Mute   bool   `json:"mute"`
	Deaf   bool   `json:"deaf"`
}

type UserModifyRequest struct {
	Premium string       `json:"premium"`
	Users   []UserModify `json:"users"`
}
