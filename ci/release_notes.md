# Improvements

- Submit buttons on forms now (a) disable themselves when clicked
  and (b) change their text to indicate an ongoing operation.
  This greatly increases the usability of the web UI.  See #505

- The default password for the failsafe account has been changed
  from `shield` to `password`, for more continuity across various
  packaging formats.  See #531

# Bug Fixes

- The MotD separator no longer displays if the MotD is empty
  or not specified.  See #530
