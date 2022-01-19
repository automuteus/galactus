package endpoint

const CaptureRoute = "/api/capture"

const AddCaptureEventPartial = "/event/add/"
const AddCaptureEventFull = AddCaptureEventPartial + "{connectCode}/{eventType}"

const GetCaptureEventPartial = "/event/get/"
const GetCaptureEventFull = GetCaptureEventPartial + "{connectCode}"

const GetCaptureTaskPartial = "/task/get/"
const GetCaptureTaskFull = GetCaptureTaskPartial + "{connectCode}"

const SetCaptureTaskStatusPartial = "/task/set/"
const SetCaptureTaskStatusFull = SetCaptureTaskStatusPartial + "{taskID}"
