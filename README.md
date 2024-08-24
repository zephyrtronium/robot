# Robot

Robot is a bot for Twitch.TV IRC that learns from people and responds to them with things that it has learned.

For project management reasons, this repository now represents a complete rewrite of Robot.
The version RobotIsBroken is running is [v0.1.0](https://github.com/zephyrtronium/robot/tree/v0.1.0).
Links that RobotIsBroken posts in chat refer to that version, *not* this one.

## What data does Robot store?
NOTE: This applies to the in-progress rewrite of Robot, not to the version currently running as RobotIsBroken. See above.

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

If you want Robot not to record your messages for any reason, simply use the `give me privacy` command.
You'll still be able to ask Robot for messages.
If you'd like the bot to learn from you again after going private, use the `learn from me again` command.
