package endpoint

const SendMessagePartial = "/sendMessage/"
const SendMessageFull = SendMessagePartial + "{channelID}"

const SendMessageEmbedPartial = "/sendMessageEmbed/"
const SendMessageEmbedFull = SendMessageEmbedPartial + "{channelID}"

const EditMessageEmbedPartial = "/editMessageEmbed/"
const EditMessageEmbedFull = EditMessageEmbedPartial + "{channelID}/{messageID}"

const DeleteMessagePartial = "/deleteMessage/"
const DeleteMessageFull = DeleteMessagePartial + "{channelID}/{messageID}"

const RemoveReactionPartial = "/removeReaction/"
const RemoveReactionFull = RemoveReactionPartial + "{channelID}/{messageID}/{emojiID}/{userID}"

const RemoveAllReactionsPartial = "/removeAllReactions/"
const RemoveAllReactionsFull = RemoveAllReactionsPartial + "{channelID}/{messageID}"

const AddReactionPartial = "/addReaction/"
const AddReactionFull = AddReactionPartial + "{channelID}/{messageID}/{emojiID}"

const ModifyUserbyGuildConnectCode = "/modify/{guildID}/{connectCode}"

const GetGuildPartial = "/guild/"
const GetGuildFull = GetGuildPartial + "{guildID}"

const GetGuildChannelsPartial = "/guildChannels/"
const GetGuildChannelsFull = GetGuildChannelsPartial + "{guildID}"

const GetGuildMemberPartial = "/guildMember/"
const GetGuildMemberFull = GetGuildMemberPartial + "{guildID}/{userID}"

const GetGuildRolesPartial = "/guildRoles/"
const GetGuildRolesFull = GetGuildRolesPartial + "{guildID}"

const UserChannelCreatePartial = "/createUserChannel/"
const UserChannelCreateFull = UserChannelCreatePartial + "{userID}"

const RequestJob = "/request/job"
const JobCount = "/totalJobs"
