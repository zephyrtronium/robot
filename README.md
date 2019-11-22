# RobotIsBroken

RobotIsBroken is a bot for IRC that learns from people and responds to them with things that it has learned. Its default configuration works for Twitch.TV chat.

# Running

robot.go is `go run`-able. Alternatively, the binary distribution works standalone. Either way, a number of configuration options exist:

 - `-server=server:port` Server and port to which to connect. The default is `irc.twitch.tv:6667`.
 - `-pass=password` Server login password. The default is no password. For Twitch.TV, use your OAuth token with chat capabilities, e.g. from [http://twitchapps.com/tmi/]. Include `oauth:` at the start, so that it looks like `-pass=oauth:longgibberishstuffhere`.
 - `-nick=nickname` Nickname to use. For Twitch.TV, this should be the login name for the account.
 - `-user=username` Username to use. For Twitch.TV, this should be the login name for the account.
 - `-real=realname` Realname to use. For Twitch.TV, this should be the login name for the account.
 - `-channel=#channel1,#channel2` Comma-separated list of channels in which to participate. On Twitch.tv, channels are of the format `#username`, e.g. BrokenCowLeg's channel is `#brokencowleg`.
 - `-listen=#channel,#channel2` Comma-separated list of channels to learn from but not to speak in.
 - `-length=<integer>` Order of the Markov chains used to generate replies. Smaller numbers require less learning before saying new things, but tend to make less sense. Values of 2 or 3 work well.
 - `-dict=file.dict` "Brain" file. Things the bot learns are stored in this file and remembered when the bot is restarted.
 - `-secret=word` Password for administrative commands.
 - `-sendprob=<value>` Default probability of generating a reply to a message, between 0 (never reply) and 1 (always reply). Values of 0.05 to 0.2 seem to work well.
 - `-ssp=#channel1=p1,#channel2=p2` Per-channel send probabilities, applied after the default set by `-sendprob`.
 - `-respond={true|false}` Whether to always respond when addressed.
 - `-ignore=user1,user2` Comma-separated list of usernames to ignore. This is helpful e.g. for ignoring NightBot and other bots.
 - `-regexignore="expression"` Regular expression of PRIVMSG arguments to ignore. The default is `^!|://|\.(com|net|org|tv|edu)|^\001`, which ignores, in order, messages beginning with `!`; containing `://`, `.com`, `.net`, `.org`, `.tv`, or `.edu`; and CTCP messages (`/me`).
 - `-admin=user1,user2` Comma-separated list of administrator usernames. Users given here can use special commands.
 - `-speed=<integer>` "Typing" speed; the bot will delay this many milliseconds per character when sending a message.
 - `-roll=<integer>` Roll queue length. See the documentation of the `roll` command below. The default value is 0, meaning the roll queue is disabled and messages are learned immediately.

It is helpful to create a batch script to run the bot on a double-click. An example might look like:

```
C:\path\to\robot.exe -server=irc.twitch.tv:6667 -nick=robotisbroken -user=robotisbroken -real=robotisbroken -channel="#brokencowleg,#robotisbroken" -listen="" -admin="zephyrtronium,brokencowleg" -ignore="robotisbroken,nightbot" -secret="]" -length=3 -dict=robot.3.dict -sendprob=0.25 -pass=oauth:gibberishstuff
```

# Commands

On non-Twitch IRC, use commands by messaging the bot directly with the secret, a space, and the command to use. On Twitch.TV, use commands by being an admin, going to the bot's chat (i.e. its profile), and messaging with the secret, a space, and the command to use. As an example, with `]` as the secret:

```
] sendprob 0.5
```

The full list of commands:

- `quit <message>` Disconnect from the server and stop running. The message is ignored on Twitch.TV, but is still required because programming is hard.
- `join #channel1 #channel2` Space-separated list of new channels to join and in which to participate.
- `listen #channel1 #channel2` Space-separated list of new channels in which to listen but not speak, unless addressed.
- `part #channel1 #channel2` Space-separated list of channels to leave.
- `nick <newnick>` Change nickname. No effect on Twitch.TV.
- `sendprob <value>` Set a new response probability. All channels in which the bot can currently speak have their response probabilities set to the new value.
- `sendprob <value> #channel1 #channel2 ...` Set a new response probability for a list of channels. Channels not listed are unaffected.
- `raw <stuff>` Send a raw message to the server. Use carefully.
- `ignore person1 person2` Space-separated list of new users to ignore.
- `unignore person1 person2` Space-separated list of users to unignore.
- `admin person1 person2` Space-separated list of users to make admins.
- `respond {on|off}` Control whether to always respond when addressed, which is when the first word of a message contains the bot's current nickname.
- `regexignore <expression>` Set a new regular expression to filter messages.
- `speed <integer>` Set a new typing speed.
- `roll <integer>` Set a new roll queue length. This is the number of messages the bot delays before actually learning. If the roll queue length is greater than zero and a Twitch user is banned or timed out, all messages within the roll queue from that user are deleted. If the bot is launched with a zero roll queue length and without caps, then the roll queue cannot be activated.
- `forget <expression>` Purge messages from the roll queue if they match the given expression. See [https://golang.org/s/re2syntax] for syntax details. Spaces in the expression are replaced with `\s+`, i.e. matching one or more whitespace characters.
- `status <kind>` Inquire about the bot's status. The argument may be `nick` to have the bot say what it believes its nick is, `req` for Twitch capabilities requested, `admin` for the current list of admins, `ignored` for the list of ignored users, `chans` for the list of channels the bot is in along with their respective send probabilities, `re` for the current regex ignore, `respond` for whether the bot responds when addressed, `roll` for the roll queue length, `speed` for the bot's simulated typing speed, `knowledge` for the prefix length and the number of prefixes and suffixes the bot knows, or `all` for all of the above in separate messages.

# Stopping

To stop the bot, press Ctrl+C at the command prompt or use the `quit` command.
