package galactus

type UserModify struct {
	UserID uint64 `json:"userID"`
	Mute   bool   `json:"mute"`
	Deaf   bool   `json:"deaf"`
}

type UserModifyRequest struct {
	Premium int16        `json:"premium"`
	Users   []UserModify `json:"users"`
}
