package plugin

/*

Hi Jaime! Here's where we define exit codes that the plugins will use, so that all plugins can behave in a consistent manner

*/

const SUCCESS = 0
const USAGE = 1
const UNSUPPORTED_ACTION = 2
const EXEC_FAILURE = 3
const PLUGIN_FAILURE = 4
const JSON_FAILURE = 10
const ENDPOINT_REQUIRED = 11
const RESTORE_KEY_REQUIRED = 12
