# History Backupper Chrome Web Store Description

History Backupper helps you keep a personal backup of your Chrome browsing history.

Chrome's built-in history is useful, but it is limited to 90 days and can be difficult to preserve, move, or analyze over time. History Backupper gives users a simple way to regularly send their browser history to a location they choose, so they can keep their own long-term record outside of Chrome.

After installation, the user opens the extension popup and enters an upload destination. The extension can then upload browsing history automatically on a configurable schedule. Users can also start an upload manually at any time.

History Backupper is useful for people who want to:

- Keep a personal backup of their browsing history
- Preserve history before changing devices or browser profiles
- Maintain their own archive for research, productivity, or personal recordkeeping
- Identify history from different computers by assigning each browser a device name
- Control where their history data is sent

The extension includes a small settings popup where users can:

- Set the upload destination
- Choose how often history is uploaded
- Set or edit a device name
- See when the last successful upload happened
- Manually upload all available history
- Manually upload only new history since the last successful upload

History Backupper only uploads to the destination the user configures. It does not provide a public cloud service or send history to a third-party service by default. Users are expected to use an upload destination they control or trust. Example Source code for an implementation of a compatible upload service is provided at https://github.com/SuhasHebbar/history-backup-extension.

The extension requests access to browser history so it can read the history items selected for backup. It also requests permission for the user's chosen upload destination so it can send the backup there. These permissions are used only for the backup behavior described above.
