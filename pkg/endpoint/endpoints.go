package endpoint

const SendMessagePartial = "/sendMessage/"
const SendMessageFull = SendMessagePartial + "{channelID}"

const SendMessageEmbedPartial = "/sendMessageEmbed/"
const SendMessageEmbedFull = SendMessageEmbedPartial + "{channelID}"

const DeleteMessagePartial = "/deleteMessage/"
const DeleteMessageFull = DeleteMessagePartial + "{channelID}/{messageID}"

const ModifyUserbyGuildConnectCode = "/modify/{guildID}/{connectCode}"

const RequestJob = "/request/job"
const JobCount = "/totalJobs"
