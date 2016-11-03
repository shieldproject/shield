## Improvements

- The `shield` CLI has new canonical forms for commands.  In a
  nutshell, `show <thing>` became just `<thing>`, `list <things>`
  turned into `<things>`, and all multi-word commands became
  hyphenated.  Check out `shield help` for details.

  Previous aliases for these commands should continue to work for
  a few releases / months.  Some time in 2017, maybe, those
  aliases will stop working. **PLEASE UPDATE YOUR SCRIPTS /
  AUTOMATION ACCORDINGLY**

## Bug Fixes

- Purge tasks will now be considered runnable and will be
  executed when schedule, instead of setting in pending forever
  and stacking up.
