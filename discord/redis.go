package discord

import "fmt"

//func TasksKey(connectCode string) string {
//	return "automuteus:tasks:code:" + connectCode
//}

func BroadcastTaskAckKey(taskID string) string {
	return fmt.Sprintf("automuteus:tasks:broadcast:ack:%s", taskID)
}

func CompleteTaskAckKey(taskID string) string {
	return fmt.Sprintf("automuteus:tasks:complete:ack:%s", taskID)
}

func TasksSubscribeKey(connectCode string) string {
	return "automuteus:tasks:subscribe:" + connectCode
}
