# Robot

Robot is a bot for Twitch.TV IRC that learns from people and responds to them with things that it has learned.

## Tools for broadcasters and mods

Robot has a number of features for managing activity level and knowledge. Most are automatic: for example, by default, the bot is configured not to send more than one message per two seconds (although this can be changed), and it deletes recently learned information from users who get banned or timed out.

There are a few [commands](#commands) for more explicit management. All of these commands require [admin priviliges](#privileges) (which are assigned automatically to the broadcaster and mods). The most relevant ones are:

- `forget pattern` deletes all recent messages containing the supplied pattern. E.g., if the bot's username is "Robot", then saying `@Robot forget anime is trash` makes the bot remove all messages received in the last fifteen minutes that contain "anime is trash" anywhere in the message.
- `you're too active` reduces the random response rate, making the bot speak less often when not addressed.
- `set response probability to nn%` sets the random response rate to a particular value. This is a good way to make the bot more talkative. Depending on the channel's activity level, the most reasonable values for this are usually somewhere around 2% to 10%.
- `be quiet for 2 hours` makes Robot neither learn from nor speak in the channel for two hours. You can use other amounts of time, but the bot limits the length to 12 hours – if you really need longer, contact the bot owner. If you don't provide an amount of time, it defaults to one hour.
- `you may speak` disables a previous use of the "be quiet" command.

For the exact syntax to use these commands, see [the relevant section](#commands).

## What information does Robot store?

Robot stores three types of information:

- Configuration details. This includes things like channels to connect to, how frequently to send messages, and who has certain [privileges](#privileges). For the most part, this information is relevant only to bot owners, broadcasters, and mods.
- Fifteen-minute history. Robot records all chat messages received in the last fifteen minutes, storing the username of the sender, the channel it was sent to, the time it was received, and the full message text. Robot uses this information to delete messages it's learned under [certain circumstances](#tools-for-broadcasters-and-mods). Whenever Robot receives a new message, all records older than fifteen minutes are removed. Robot also records the messages it's sent in the last fifteen minutes.
- Markov chain tuples. This is the majority of Robot's data, a simple list of prefix and suffix words tagged with the location that prefix and suffix may be used. This data is anonymous; Robot does not know who sent the messages that were used to obtain this information.

If you want Robot not to record information from you for any reason, contact the bot owner asking to be ignored. Ask the broadcaster how to reach the bot owner if you aren't sure. You may lose the ability to use Robot's commands, including generating messages.

## How Robot works

Robot uses the mathematical concept of Markov chains, extended in some interesting ways, to learn from chat and apply its knowledge. Here's an example.

Let's say Robot receives this chat message: `Can you provide a better example for me please?` The first thing it will do is run some preliminary checks to make sure it's ok to learn from the message, e.g. no links, sender isn't a bot, &c.

This particular message is fine. Robot's next step is to break it up into a list of tokens – basically words, except that the English articles "a," "an," and "the" are usually combined with the next word as well, along with invisible tokens for the start and end of a message. The tokens here are `Can`, `you`, `provide`, `a better`, `example`, `for`, `me`, `please?`.

Robot is configured with an "order", a number governing how much context matters when learning from messages. Let's say in this case that the bot has an order of 4. This means Robot takes groups of five tokens at a time and learns that the first four can be followed by the fifth. So, the bot will learn:

- `Can` may be used to start a message.
- `you` may follow `can` at the start. (Robot learns the exact capitalization of the "to" word, but ignores it for the "from" words.)
- `provide` may follow `can` `you` at the start.
- `a better` may follow `can` `you` `provide` at the start.
- `example` may follow `can` `you` `provide` `a better`. (At this point, the start of the message is more than four tokens old, so she doesn't consider it anymore.)
- `for` may follow `you` `provide` `a better` `example`.
- `me` may follow `provide` `a better` `example` `for`.
- `please?` may follow `a better` `example` `for` `me`.
- A message may end after `example` `for` `me` `please?`.

Learning the message is finished. But, robots don't like learning things they'll never use.

When it's time for Robot to think of something to say, the bot does a "random walk" on everything it's learned. Starting with the invisible token for the end of a message, the bot picks out everything it knows can follow, then chooses one of those words entirely at random. Let's say it picks the word `You`. Robot records that the random walk went to `You`, then looks for everything that can follow `you` at the start. It might pick `SHOULD`; record it and look from `you` `should`, and maybe choose `HAVE`; then `waited`.

Now that the walk is at four words chosen, same as the order, Robot stop caring about the start again, so it's looking for words that can follow `you` `should` `have` `waited` seen anywhere in a message. This might find `till`; then next it'll look for words following `should` `have` `waited` `till`. Say it finds `4am`. It'll look for things that can follow `have` `waited` `till` `4am`, and we'll say it finds the invisible end-of-message token. So, the generated message is `You SHOULD HAVE waited till 4am`.

(This example is a bit simplified because the database I have while writing this is still a bit small. Robot has some clever tricks to try to find more original ~~nonsense~~ messages that I didn't talk about, e.g. varying the number of prefix words that need to match when finding a suffix.)

## Commands

Robot acknowledges chat messages which start or end with the bot's username, ignoring case, possibly preceded by an `@` character or followed by punctuation. For example, if the bot's name is "Robot", then it will recognize these as command messages:

- `@Robot madoka`
- `madoka @rObOt`
- `robot madoka`
- `Robot: madoka`
- `madoka Robot ?`

These are *not* recognized as commands:

- `madoka @Robot homura`
- `¡Robot madoka!`

When Robot recognizes a command, it strips the triggering portion from the message, and the remainder is the command invocation. So, in all of the command examples above, the command invocation is `madoka`. If a message both starts and ends by addressing Robot, only the start is removed for the invocation; e.g. the invocation for `@Robot madoka @Robot` is `madoka @Robot`.

The command invocation is checked against the list of commands for which the user who sent the command message has appropriate [privileges](#privileges). Robot is designed to understand some amount of English for command invocations; there are usually multiple forms you can use to perform a given command.

### Commands for everyone

- `where is your source code?` provides a link to this page, along with a short summary of prominent technologies leveraged.
- `generate something with starting chain` generates a message that starts with `starting chain`. (Nothing happens if the bot doesn't know anything to say from there.)
- `uwu` genyewates an especiawwy uwu message.

If a command invocation doesn't match any command, it instead prompts Robot to speak.

### Commands for admins

- `forget <pattern to forget>` un-learns all recent messages that contain the text "pattern to forget".
- `help <command-name>` displays a brief help message on a command.
- `invocation <command-name>` displays the exact regular expression used to match a command.
- `list commands` displays all admin and regular command names.
- `be quiet until tomorrow` causes the bot to neither learn from nor randomly speak in the channel for twelve hours.
- `be quiet for 2 hours` is the same, but for 2 hours, or any other duration.
- `you may speak` undoes "be quiet" commands, allowing the bot to learn and speak again immediately.
- `you're too active` reduces Robot's random response rate.
- `set response probability to <prob>%` sets Robot's random response rate to a particular value.
- `speak <n> times` generates up to n messages at once, bypassing the bot's rate limit. The maximum for n is 5.
- `raid` generates five messages at once.

## Privileges

Robot has five privilege levels:

- `owner` is a special privilege level for the [bot owner](#running-your-own-instance).
- `admin` gives access to extra commands for moderating robot's activity levels and knowledge.
- `regular` is the default privilege level, for basic fun with the Markov chain features.
- `ignore` removes access to any commands, including Markov chain features. Robot also does not learn from ignored users.
- `bot` is a mix of admin and ignore privileges. Users with bot privileges can invoke admin commands, but Robot does not learn from their other messages.

Robot recognizes scans a user's chat badges to assign default privileges. Unless overridden per user, broadcasters and mods, Twitch staff have owner privileges, and everyone else (including VIPs and subscribers) has regular privileges.

## Rate limits

When Robot wants to generate a message, whether randomly or through a command, the bot requests a ticket from its rate limiter for the channel it would send to. If there is no ticket available, Robot does not generate the message.

There are two knobs on Robot's rate limiters: the "rate" and the "burst size." I have done a poor job of explaining what these mean elsewhere, but essentially, the burst size is the maximum number of tickets Robot can take, and the rate is how many tickets regenerate per second.

Say a channel has a rate of 0.1 and a burst size of 2. Robot hasn't said anything for a couple minutes. Someone asks the bot to generate a message at 10:53:00; Robot takes a ticket, leaving two remaining, and sends a message. At 10:53:02, another person talks and triggers a random message; Robot takes another ticket, and now there are 1.2 tickets. Another person demands an uwu at 10:53:07, leaving 0.7 tickets. Someone in the channel needs a meme at 10:53:10; the rate limiter has *just* regenerated a full ticket, which Robot takes, leaving 0.0. Then, at 10:53:19, another person asks for an uwu, because they're beautiful, but the rate limiter has only 0.9 tickets, so Robot cannot take one and so does not speak.

In addition to the rate limiter described above, Robot has a global rate limiter that prevents her from trying to send more than 100 messages per thirty seconds on average, per Twitch's documentation. However, for that global limiter, she waits for a ticket to become available instead of giving up if there isn't one already.

## Running your own instance

Robot is designed to be reasonable to install and use on your own. Doing so lets you control exactly when the bot runs and what information is in its database. If you can program in Go and SQL, you can even modify how Robot works to add new features or to remove existing ones, as long as you follow the [GPLv3 license](COPYING).

### Installing and running

First, make sure you have the latest versions of Go and GCC installed. If you open a command prompt or terminal, entering `go version` should print something like `go version go1.15.2 windows/amd64`, and entering `gcc --version` should print something like `gcc.exe (tdm64-1) 9.2.0`. If either of these commands fails, follow the installation instructions for [Go](https://golang.org/doc/install) and, on Windows, [TDM-GCC](https://jmeubank.github.io/tdm-gcc/download/). (If you aren't on Windows, GCC is almost certainly installed already.)

Also recommended is to install an SQLite3 database interface, like [SQLiteStudio](https://sqlitestudio.pl/).

With at least Go and GCC installed, simply enter `go get github.com/zephyrtronium/robot/...`. This installs the `robot`, `robot-convert`, `robot-init`, and `robot-talk` commands. At this point, you should be able to enter `robot -help` to see a basic help message.

Before you can run Robot, you'll need to use `robot-init` to initialize a database. You'll probably want to copy and modify [the example configuration](cmd/robot-init/example.json), then do `robot-init -conf modified.json -source robot.sqlite3`. See [the README for `robot-init`](cmd/robot-init/README.md) for more information.

You'll also need an OAuth token for Twitch, with at least `chat:read` and `chat:edit` scopes. If you don't already have one, you can get one through [the TMI OAuth generator](https://twitchapps.com/tmi/).

Finally, run Robot using `robot -source robot.sqlite3 -token y0UrOAuth70Ken`.

### Database structure

Robot's database tables are:

- `chans` - configuration for channels known to the bot.
	+ `name` - primary key, name of the channel; must begin with '#' and be all lower case, otherwise Twitch will silently ignore the bot trying to join.
	+ `learn` - tag to use for learned messages
	+ `send` - tag to select from to generate messages
	+ `lim` - maximum length of generated messages
	+ `prob` - probability that a non-command message will trigger generating a message; must be between 0 and 1
	+ `rate` - maximum average messages per second to send
	+ `burst` - maximum number of messages to send in a burst
	+ `block` - regular expression matching messages to block, in addition to the block expression in `config`
	+ `respond` - whether commands can generate messages, separate from the random speaking chance
	+ `silence` - datetime before which the bot will not learn or speak
- `config` - global configuration, only row 1 is used
	+ `me` - bot's username, used as nick
	+ `pfix` - Markov chain order, as described [above](#how-robot-works)
	+ `block` - regular expression matching messages to block
- `emotes` - global and per-channel emotes to append to messages
	+ `tag` - send tag where used, or everywhere if null
	+ `emote` - emote text
	+ `weight` - integer weight; higher values mean more likely to select
- `generated` - messages generated in the last fifteen minutes
	+ `time` - timestamp of generated message
	+ `tag` - tag used to generate the message
	+ `msg` - generated message text
- `history` - messages learned from in the last fifteen minutes
	+ `tid` - Twitch IRC message ID
	+ `time` - timestamp of message receipt
	+ `sender` - username of the message sender
	+ `chan` - channel received in
	+ `tag` - tag used to learn the message
	+ `msg` - message text
- `privs` - global and per-channel user priviliges
	+ `user` - username receiving this privilege
	+ `chan` - channel where applicable, or NULL if a global default
	+ `priv` - privilege type, one of "owner", "admin", "bot", or "ignore". (Regular privs are implied by not being in the table, or can be forced by setting this to the empty string.)
- `quotes` - unused
- `tuplesn`, where `n` is a number - Markov chain data n prefix words
	+ `tag` - tag with which this chain was learned
	+ `p0`, `p1`, ... - prefix words; null means before start of message
	+ `suffix` - suffix word; null means end of message

### Owner commands

- `warranty` prints a brief "NO WARRANTY" message, extracted from the GPLv3, on the bot's terminal.
- `disable <command-name>` globally disables a command.
- `enable <command-name>` globally re-enables a command.
- `resync` synchronizes the bot's channel configurations with what is in the database. Use this after modifying the chans, emotes, or privs tables.
- `EXEC <query>` executes an arbitrary mutating SQL query.
- `raw <command> <params> :<trailing>` sends a raw IRC message to the server.
- `join <channel> [<learn-tag> <send-tag>]` joins a channel and adds its configuration to the database. If the tags are not given, the inserted values will be NULL.
- `give <user> {owner|admin|bot|regular|ignore} privileges [in <channel>|everywhere]` sets a user's privileges level, in the current channel if omitted.
- `quit` disconnects from the IRC server and closes the bot.
- `reconnect` disconnects from and then reconnects to the IRC server.
- `list commands` (overriding the admin version) lists all commands, including owner-only and disabled ones. The latter are marked by a `*`.
