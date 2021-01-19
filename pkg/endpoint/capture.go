package endpoint

const CaptureRoute = "/api/capture"

const JobCount = "/totalJobs"
const RequestJob = "/request/job"

const AddCaptureEventPartial = "/addCaptureEvent/"
const AddCaptureEventFull = AddCaptureEventPartial + "{connectCode}/{eventType}"

const GetCaptureEventPartial = "/getCaptureEvent/"
const GetCaptureEventFull = GetCaptureEventPartial + "{connectCode}"

const GetCaptureTaskPartial = "/getCaptureTask/"
const GetCaptureTaskFull = GetCaptureTaskPartial + "{connectCode}"

const SetCaptureTaskStatusPartial = "/setCaptureTaskStatus/"
const SetCaptureTaskStatusFull = SetCaptureTaskStatusPartial + "{taskID}"
