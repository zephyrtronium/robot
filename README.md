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
 	- `-sendprob=<value>` Probability of generating a reply to a message, between 0 (never reply) and 1 (always reply). Values of 0.25 to 0.35 seem to work well.
 	- `-ignore=user1,user2` Comma-separated list of usernames to ignore. This is helpful e.g. for ignoring NightBot and other bots.
 	- `-admin=user1,user2` Comma-separated list of administrator usernames. Users given here can use special commands.

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
	- `listen #channel1 #channel2` Space-separated list of new channels in which to listen but not speak.
	- `part #channel1 #channel2` Space-separated list of channels to leave.
	- `nick <newnick>` Change nickname. No effect on Twitch.TV.
	- `sendprob <value>` Set a new response probability.
	- `raw <stuff>` Send a raw message to the server. Use carefully.
	- `ignore person1 person2` Space-separated list of new users to ignore.
	- `unignore person1 person2` Space-separated list of users to unignore.
	- `admin person1 person2` Space-separated list of users to make admins.

# Stopping

To stop the bot, press Ctrl+C at the command prompt.

Robot does not understand the concept of network errors (despite how much I tried to get them to be understood) and will hang indefinitely when they happen. You'll have to manually stop and restart the bot whenever your internet (or the server's) fails.

# Linux

The bot requires some modification to run on non-Windows. Specifically, lines 243-246 (marked "WINDOWS-DEPENDENT") and lines 356-361 (starting with `case <-isig:`) must be commented out, and lines 362-369 (starting with `case isig := <-signal.Incoming:`) must be uncommented.
