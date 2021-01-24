package endpoint

const DiscordRoute = "/api/discord"

const DiscordJobCount = "/job/count"
const DiscordJobRequest = "/job/request"

const SendMessagePartial = "/message/send/"
const SendMessageFull = SendMessagePartial + "{channelID}"

const SendMessageEmbedPartial = "/messageEmbed/send/"
const SendMessageEmbedFull = SendMessageEmbedPartial + "{channelID}"

const EditMessageEmbedPartial = "/messageEmbed/edit/"
const EditMessageEmbedFull = EditMessageEmbedPartial + "{channelID}/{messageID}"

const DeleteMessagePartial = "/message/delete/"
const DeleteMessageFull = DeleteMessagePartial + "{channelID}/{messageID}"

const RemoveReactionPartial = "/reaction/remove/"
const RemoveReactionFull = RemoveReactionPartial + "{channelID}/{messageID}/{emojiID}/{userID}"

const RemoveAllReactionsPartial = "/reaction/remove/all/"
const RemoveAllReactionsFull = RemoveAllReactionsPartial + "{channelID}/{messageID}"

const AddReactionPartial = "/reaction/add/"
const AddReactionFull = AddReactionPartial + "{channelID}/{messageID}/{emojiID}"

const ModifyUserPartial = "/user/modify/"
const ModifyUserFull = ModifyUserPartial + "{guildID}/{connectCode}"

const GetGuildPartial = "/guild/get/"
const GetGuildFull = GetGuildPartial + "{guildID}"

const GetGuildChannelsPartial = "/guild/channels/get/"
const GetGuildChannelsFull = GetGuildChannelsPartial + "{guildID}"

const GetGuildMemberPartial = "/guild/member/get/"
const GetGuildMemberFull = GetGuildMemberPartial + "{guildID}/{userID}"

const GetGuildRolesPartial = "/guild/roles/get/"
const GetGuildRolesFull = GetGuildRolesPartial + "{guildID}"

const UserChannelCreatePartial = "/user/channel/create/"
const UserChannelCreateFull = UserChannelCreatePartial + "{userID}"

const GetGuildEmojisPartial = "/guild/emojis/get/"
const GetGuildEmojisFull = GetGuildEmojisPartial + "{guildID}"

const CreateGuildEmojiPartial = "/guild/emoji/create/"
const CreateGuildEmojiFull = CreateGuildEmojiPartial + "{guildID}/{name}"
