package endpoint

const RequestJob = "/request/job"
const JobCount = "/totalJobs"

const AddCaptureEventPartial = "/addCaptureEvent/"
const AddCaptureEventFull = AddCaptureEventPartial + "{connectCode}/{eventType}"

const GetCaptureTaskPartial = "/getCaptureTask/"
const GetCaptureTaskFull = GetCaptureTaskPartial + "{connectCode}"

const SetCaptureTaskStatusPartial = "/setCaptureTaskStatus/"
const SetCaptureTaskStatusFull = SetCaptureTaskStatusPartial + "{taskID}"

const GetGuildAMUSettingsPartial = "/getAMUSettings/"
const GetGuildAMUSettingsFull = GetGuildAMUSettingsPartial + "{guildID}"
