# Character Profile Plugin for Mattermost

This plugin allows Mattermost users to use several profile pictures and display names.

## Manual installation

```bash
git clone https://github.com/svenssonaxel/mattermost-plugin-character-profiles.git
cd mattermost-plugin-character-profiles/
export MM_SERVICESETTINGS_SITEURL=https://your.mattermost.site
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

## Usage

See [helptext.md](server/helptext.md) or type `/character help` in your Mattermost client.
