# The `/character` command
Sometimes you may want to write messages that appear to have been sent by someone other than yourself, such as when speaking as your PC or an NPC. The `/character` command provides this functionality. Here is how to use it:

## Manage character profiles
In order to act as several PCs or NPCs, you create a "character profile" for each. Character profiles that you create can be used in any channel in any team, but only by you. A character profile has an identifier that can only be lowercase a-z, a display name that can be anything you like, and optionally a profile picture. As a special case, identifiers `myself` and `me` refer to your real Mattermost profile.
- `/character haddock=Captain Haddock`: Create a character profile with identifier `haddock` unless it already exists, and set its display name to `Captain Haddock`.
- `/character picture haddock=Captain Haddock`: Create a character profile with identifier `haddock` unless it already exists, set its display name to `Captain Haddock`, and set its profile picture to the picture uploaded in the parent message. (Note that you can **not** attach a picture to the slash command itself, for technical reasons.)
- `/character picture haddock`: Modify an existing character profile by updating the profile picture to the picture uploaded in the parent message, leaving the display name as it is. (Note that you can **not** attach a picture to the slash command itself, for technical reasons.)
- `/character delete haddock`: Delete character profile with identifier `haddock`.
- `/character list`: List your character profiles.

## Set a default character profile
Sometimes, e.g. for PCs, you want to use a certain character profile for most messages. For each channel, you can set a default character profile identifier that will be used for all messages except those sent using the single message functionality described below.
- `/character I am haddock`: Set default character profile identifier for the current channel to `haddock`.
- `/character I am myself`: Remove the default character profile for the current channel.
- `/character who am I`: List default character profiles for the channels in this team.

## Use a character profile for a single message
Sometimes, e.g. for NPCs, you want to use character profiles in a one-off fashion. To do so, prefix your message with the character profile identifier, followed by a colon, followed by either a space or a newline. If you have a character profile with that identifier, it will be applied to the message and the prefix will be removed.
- `haddock: Pock-marked pin-headed pirate of a pilot!`: Send a one-off message using character profile identifier `haddock`. The message will show with display name `Captain Haddock`.
- `me: I apologize for Captain Haddock's language.`: Send a one-off message using your real Mattermost profile.

## Limitations
Changes to character profiles are only applied to messages created or edited after the change. When you edit and save a message, it will use the same profile identifier as when originally sent (or when last edited). If you want to change it, you can prefix the message to use the single message functionality described above. Making changes to character profiles will not change how past messages appear, unless edited after the changes are made. Setting default character profile identifier will never affect message editing.
