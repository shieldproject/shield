# Improvements
- When the WebUI prints target/store configuration, it will now pretty-print the JSON.
- When running an adhoc backup job, or restore, the CLI and APIs
  now return the UUID of the created task, to allow users and scripts to
  more easily identify what task was created to watch its logs.
- `make dev` and the `testdev` script received a number of enhancements for
  making life easier with developing shield.
 
# Bug Fixes

- Fixed an issue where authorized headers weren't being properly passed
  if a user had not been authenticated prior to sending a PUT/POST request
  to the API.
- Fixed issue where the `exclude` option to the `fs` plugin's backups was
  including instead of excluding.
