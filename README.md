# Robot

Robot is a bot for Twitch.TV IRC that learns from people and responds to them with things that it has learned.


## Tools for broadcasters and mods

Robot has a number of feaatures for managing activity level and knowledge.
Generally speaking, the bot is designed to treat chat moderation actions as moderating its knowledge as well.
This includes:

- When an individual message is deleted, Robot forgets anything it learned from it.
- When a chatter is banned or timed out, Robot forgets anything it learned from that chatter in the last fifteen minutes.
- When one of Robot's messages is deleted, Robot forgets every message that was used to generate it.
- When Robot is timed out, it forgets every message used to generate everything it sent in the last fifteen minutes.
  (Please do not ban Robot. The bot owner will likely be shadowbanned as well, and the bot won't rejoin after being unbanned.)
- Robot doesn't learn from chat while the stream is offline.

In addition to the above, Robot provides explicit moderation [commands](#commands).

- `forget pattern` tells robot to forget every message in the last fifteen minutes containing the supplied pattern.
  E.g., if the bot's username is "Robot", then saying `@Robot forget anime is trash` makes the bot remove all messages in the last fifteen minutes that contain "anime is trash".
- As a special case, `forget everything` causes Robot to remove all messages in the last fifteen minutes, regardless of content.


## What data does Robot store?

Robot learns from most messages in the Twitch chats she's in while the stream is online.
In order to ensure moderators are able to do their job, she stores additional metadata on those messages as well.
Here is the complete list of information types that go into Robot's databases:

- Message metadata.
  This includes the message ID, channel, timestamp, and an obscured representation of the message sender (described below).
  Robot uses this for moderation purposes, e.g. to delete what she's learned from people who are banned.
- Markov chain tuples.
  This is the majority of Robot's data, a list of prefixes and the words that can follow them.
  Each tuple is associated with an entry from the message metadata.
- A list of users who have opted out of message collection.
  This applies globally; there is no way to tell where the user asked for this.
- Messages Robot has produced with the message IDs used to produce them and some additional info for analytics.
  No data collected from users is here, except insofar as the messages are produced from things people have said.

In the message metadata, the message sender is stored using a cryptographic hash of the sender's user ID, the channel it was sent to, and the fifteen-minute time period in which it was sent.
Roughly speaking, if Robot has been learning from Bocchi, message metadata together with Markov chain tuples *can* answer questions like these:

- What are all the messages Robot has learned from Bocchi in the last thirty minutes?
- Did Bocchi talk in KessokuBand's chat on 21 Feb 2024 between 0900 and 1000 while KessokuBand's stream was online? (Robot never learns from offline streams.)
- Was Bocchi the person who sent this particular message that Robot learned in KessokuBand's chat?

On the other hand, it is infeasible or at least very expensive for Robot's data to answer questions like these:

- Who are all the people Robot has learned from in the last thirty minutes?
- What times has Bocchi been active in KessokuBand's chat?
- Who sent this particular message that Robot learned in KessokuBand's chat?
- What channels has Bocchi been active in?

If you want Robot not to record your messages for any reason, simply use the `give me privacy` [command](#commands).
You'll still be able to ask Robot for messages and use other commands.
If you'd like the bot to learn from you again after going private, use the `learn from me again` command.


## How Robot works

Robot uses the mathematical concept of Markov chains, extended in some interesting ways, to learn from chat and apply its knowledge.
Here's an example.

Let's say Robot sees this chat message: `Bocchi the Rock!`.
The first thing it will do is run some preliminary checks to make sure it's ok to learn from the message,
e.g. no links, sender hasn't opted out, &c.

This particular message is fine.
Robot's next step is to break it up into a list of *tokens* – basically words or stretches of non-letter characters followed by spaces.
The tokens here are `<beginning of message>`, `Bocchi `, `the `, `Rock`, `!`, `<end of message>`.
The "beginning of message" and "end of message" are invisible tokens that are always there, at least conceptually.

For each token, Robot now learns that all of the ones before it, as a group, can be followed by the one after.
The *prefix* is made lowercase for this to help improve variety later.
That is to say:

- `Bocchi ` can comme at the start of the message.
- `the ` can come after `bocchi ` at the start.
- `Rock` can come after `the ` after `bocchi ` at the start.
- `!` can come after `rock` after `the ` after `bocchi ` at the start.
- After all of the above, the message can end.

Learning the message is finished.
But robots don't like learning things they'll never use.

When it's time for Robot to think of something to say, the bot does a *random walk* on everything it's learned.
Starting with the invisible beginning-of-message token, the bot picks out everything it has learned can follow and picks one option at random.

Let's say it picks the word `You `.
Robot records that the random walk went to `You `, then looks for everything that can follow `<beginning of message>` `you ` (converted to lowercase, as during learning).
It might pick `SHOULD ` next; record it and look from `<beginning of message>` `you ` `should `, and maybe choose `HAVE `; then `waited `.

To make generated messages more interesting, Robot can also shorten the length of the context it's using to search when there are few options.
Let's say after `waited ` it starts applying this technique.
Instead of looking for `<beginning of message>` `you ` `should ` `have ` `waited `, it drops the beginning token and tries again for messages that contained "you should have waited" anywhere, rather than only at the start.
This might still not help much, so it does it again, and we'll say once more, so that now it's looking for ~~`<beginning of message>`~~ ~~`you `~~ ~~`should `~~ `have ` `waited `.

Now it finds `so ` as the next token.
Along with adding it to the random walk, it restores one of the tokens it dropped from the random walk, in case that will match something else it's learned.
So the next search happens with `should ` `have ` `waited ` `so `.
Next it picks `long`, followed by `! ` and `<end of message>`.
So, the generated message is `You SHOULD HAVE waited so long!`.


## Commands

Robot understands commands to be messages which start or end with the bot's username, ignoring case, possibly preceded by an `@` character and possibly followed by punctuation when at the start.
For example, if the bot's username is "Robot", then it will recognize these as commands:

- `@Robot bocchi`
- `bocchi @rObOt`
- `robot bocchi`
- `Robot: bocchi`

These are *not* recognized as commands:

- `bocchi Robot?`
- `bocchi @Robot kita`
- `¡Robot bocchi!`

### Commands for everyone

- `give me privacy` has Robot stop learning from your messages.
- `learn from me again` undoes `give me privacy`.
- `what information do you collect on me?` provides a link to the [section on privacy](#what-data-does-robot-store) on this page.
- `will you marry me?` asks Robot to be your waifu, husbando, or whatever other label for a domestic partner is appropriate. Robot is choosy and capricious.
- `how much do you like me?` asks Robot to compute your affection score.
- `OwO` genyewates an especiawwy uwu message.
- `how are you?` AAAAAAAAA A AAAAAAA AAA AAAA AAAAAAAA AA AA AAAAA.
- `roar` makes the bot go rawr ;3
- `where is your source code?` provides a link to this page.
- `who are you?` gives a short self-description.
- `generate bocchi` or `say bocchi` tells Robot to generate a message using `bocchi` as the prompt. (Nothing happens if the bot doesn't know anything to say from there.)

### Commands for moderators

- `echo bocchi` causes Robot to say `bocchi`, or whatever other message you give.
- `talk about ranked competitive marriage` gives a short description of Robot's marriage system.
- `forget bocchi` causes Robot to forget everything she's learned from messages containing `bocchi` in the last fifteen minutes. As a special case, `forget everything` tells her to forget all messages in the last fifteen minutes.


## Effects

Robot sometimes applies special effects to copypasta and randomly generated messages.
Effects can be configured per channel.
The possible effects are:

- `OwO` twansfowms a message using the OwO command.
- `o` roplocos vowols woth o.
- `AAAAA` AAAAA AAA AAAAAAA A. (This effect tends to trigger Twitch's spam detection, preventing the message from sending. It is typically disabled for random messages.)
