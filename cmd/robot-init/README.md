# robot-init

robot-init intializes and reconfigures databases for Robot:
```
Usage of robot-init:
  -conf string
        config JSON file
  -source string
        SQL database source
```

## Config file

robot-init reads configuration from a JSON file containing a single object. See [example.json](example.json) for an example. Below is the full description of what robot-init reads from the JSON.

robot-init attempts to make changes incrementally, so that only fields listed in the JSON file are modified. Exceptions to this rule are noted below.

### Top level object

The top level describes global configuration options. Fields are:

- `me`: Twitch username for the bot. If this is provided, then `prefix` also must be.
- `prefix`: Markov chain order, i.e. the number of prefix words per chain. If this is provided, then `me` also must be.
- `block`: Regular expression used to block messages across all channels. Omitting this leaves the current expression.
- `chans`: [See below.](#channels)
- `emotes`: A list of globally used [emotes](#emotes). Omitting this leaves the current global emotes; otherwise, the existing ones are overwritten.
- `privs`: A list of global default [privileges](#privileges). Omitting this leaves the current global privileges; otherwise, the existing ones are overwritten.

### Channels

The `chans` field of the top level is an object mapping channel names as keys to further objects of the structure below. The "default value" refers to the value used when a channel is created by robot-init but not given an explicit value for the respective field.

- `learn`: Tag used for chains learned from the channel. Omitting this leaves the current value. Setting this to the empty string disables learning in the channel. The default value is the empty string.
- `send`: Tag used when generating messages for the channel. Omitting this leaves the current value. Setting this to the empty string disables sending in the channel. The default value is the empty string.
- `lim`: Maximum length of messages generated for the channel, not including the appended emote. Omitting this or assigning a value of 0 leaves the current value. The default value is 511.
- `prob`: Probability that a non-command message will trigger generating a message. Must be between 0 and 1. Omitting this leaves the current value. The default value is 0.
- `rate`: Maximum average messages per second. See the section on rate limits in the Robot README. Omitting this or assigning a value of 0 leaves the current value. The default value is 0.5 (regenerating one use per two seconds).
- `burst`: Maximum message burst size. See the section on rate limits in the Robot README. Omitting this or assigning a value of 0 leaves the current value. The default value is 1 (one use is available between regenerations).
- `block`: Extra regular expression of messages to block in this channel. This applies in addition to the global one, not instead of it. Omitting this or assigning the empty string leaves the current expression. The default value is an expression matching no strings.
- `respond`: Whether to enable regular commands that generate messages in this channel. Omitting this leaves the current value. The default value is false.
- `silence`: Time before which to refrain from learning or randomly talking. Omitting this leaves the current value. The default value is a time in the past.
- `emotes`: A list of extra [emotes](#emotes) to use in the channel. Omitting this leaves the current channel-specific emotes; otherwise, the existing ones are overwritten.
- `privs`: A list of extra [privileges](#privileges) to use in the channel. Omitting this leaves the current channel-specific user privileges; otherwise, the existing ones are overwritten.

### Emotes

The `emotes` field of the top level and of each channel object is a list (i.e. JSON array) of these objects:

- `e`: Emote text.
- `w`: Weight. This must be a positive integer.

Both fields are required.

### Privileges

The `privs` field of the top level and of each channel object is a list (i.e. JSON array) of these objects:

- `user`: Username to which the privilege level applies.
- `priv`: Privilege level, one of "owner", "admin", "bot", "privacy", "ignore". (To set a privileged user back to regular, include the `privs` field at the appropriate scope, but leave them out of the list. All existing privileges of the same scope are overwritten when any list is provided.)

Both fields are required.
