# Improvements

* Added a /v2/mbus/status API endpoint that returns metrics about the state of
	the event message bus. This allows for some introspection into what websocket
	connections may be doing at that moment.
* The UI now has less wasted space when displaying a bunch of cards for data
	systems.
* Implemented a configurable timeout when making SSH connections to agents. A
	lower default now also keeps dead agents from taking up large amounts of
	scheduler worker time.
* SHIELD now tracks when the last time an agent erred was.
* Hidden agents are now sorted under a separate header in the web UI.
* Hiding, showing, and deleting agents can now be done from the CLI.


# Bug Fixes

* We no longer leak the file descriptors and goroutines for detached websocket
	clients
* Workers can no longer be starved out when sending events to the message bus
	if the receiver of the message bus is misbehaving because these event sends
	are now asynchronous.
* Fixed a bug where a worker could derefence a nil pointer when certain
	database selects returned no rows.
* The database layer now has more stringent locking, which both avoids certain
	threads locking each other out in SQLite, and also makes certain series of
	database operations effectively atomic.
* A couple of fixups would deadlock themselves out of the database and prevent
	fixups from actually running. Now they don't.
* Fixups now only run once instead of on every startup, like nature intended.
* The agent "Last Checked At" timestamp was being updated when the task was
  pulled off the scheduler, whether or not the agent was actually checked
  (due to other potential errors).
* Named a fixup without a name.
* Agents that failed their status checks are now once again marked as such.

