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

var PremiumBotConstraints = map[int16]int{
	0: 0,
	1: 0,   //Free and Bronze have no premium bots
	2: 1,   //Silver has 1 bot
	3: 3,   //Gold has 3 bots
	4: 10,  //Platinum (TBD)
	5: 100, //Selfhost; 100 bots(!)
}
