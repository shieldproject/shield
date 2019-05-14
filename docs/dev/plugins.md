SHIELD Plugins
==============

(*Note:* a ton more stuff needs to go here)

Plugin Configuration Metadata
-----------------------------

Plugins have configuration, and that configuration has to conform
to what the plugin code is expecting to find, or nothing works.

To make it easier for operators to properly configure their backup
jobs, each plugin executable needs to be able to describe what
information it needs, provide hints on where one might find that
information, explain what format that information must be in, etc.

This is called **plugin configuration metadata**

It looks something like this:

    [
      {
        "mode"     : "target",
        "name"     : "base_dir",
        "type"     : "abspath",
        "title"    : "Base Directory",
        "help"     : "Absolute path of the directory to backup.",
        "example"  : "/srv/www/htdocs",
        "required" : true
      },
      {
        "mode"     : "store",
        "name"     : "base_dir",
        "type"     : "abspath",
        "title"    : "Base Directory",
        "help"     : "Where to store the backup archives, on-disk.  This must be an absolute path, and the directory must exist.",
        "example"  : "/var/store/backups",
        "required" : true
      },
      {
        "mode"     : "target",
        "name"     : "include",
        "type"     : "string",
        "title"    : "Files to Include",
        "help"     : "Only files that match this pattern will be included in the backup archive.  If not specified, all files will be included."
      },
      {
        "mode"     : "target",
        "name"     : "exclude",
        "type"     : "abspath",
        "title"    : "Files to Exclude",
        "help"     : "Files that match this pattern will be excluded from the backup archive.  If not specified, no files will be excluded."
      },
      {
        "mode"    : "both",
        "name"    : "bsdtar",
        "type"    : "abspath",
        "title"   : "Path to `bsdtar` Utility",
        "help"    : "Absolute path to the `bsdtar` utility, which is used for reading and writing backup archives.",
        "default" : "/var/vcap/packages/shield/bin/bsdtar"
      }
    ]

(example taken from the `fs` plugin)

Each element in the list is a map describing exactly one field.
The order of the fields is significant; the SHIELD UIs will honor
it when rendering its form (web UI) or asking questions (CLI).

The following fields are valid for a field definition:

#### mode

One of either `target`, `store`, or `both`, indiciating which mode
of operation this field metadata definition applies to.  Some
fields are used only when a plugin is being used as a data target,
while other fields are used differently, and may have different
semantics, or different default values, when used during storage
operations vs. target system mode.

This field is required.

#### name

The internal name of the field.  This will be used to generate the
endpoint configuration.  It will never be shown to the end user.

This field is required.

#### type

The type of data, indicating what type of form field should be
presented to the user.  Valid types are:

- **string** - A (usually short) single-line string.  For the web
  UI, this would be displayed as an `<input type="text">` HTML
  form field, and the CLI might read from standard input until a
  newline is reached.

- **text** - A multi-line string  For the web UI, this is
  displayed as a `<textarea>` HTML form field, and the CLI might
  allow reading from a file, or open up an editor.

- **bool** - A yes/no proposition, sent to the backend as a
  boolean true / false value.  The web UI will display this as a
  checkbox; the CLI will ask a "yes/no" question.

- **enum** - An enumerate field, whose values must be members of a
  closed set of text strings.  The allowable values are specified
  in the `enum` key.

- **password** - Like "text", but sensitive.  Whatever the user
  enters into this field should not be visible.

- **port** - A numeric value that must be greater than 0 and less
  than 65536, for use in TCP or UDP port number configuration.

- **asbpath** - A string that represents an absolute path on a
  compatible UNIX filesystem.  Absolute paths _must_ begin with a
  forward slash `/`).

- **pem-x509** - A text field that ought to contain an X.509
  public certificate.  Additional validation can be carried out in
  UI / CLI contexts, to ensure that the proposed value _looks_
  like an X.509 PEM-encoded certificate.

- **pem-rsa-pk** - A text field that ought to contain an RSA
  private key.  The Web UI / CLI can then redact this properly, as
  well as validate that it contains the right BEGIN / END markers.

This field is required.

#### title

A display name of the field.  UIs can use this to prompt the user,
either via forms or CLI.  This value will almost always be sent to
the end user.

This field is required.

#### required

A boolean that determines whether or not the end user can leave
this field blank.  Blank fields will _not_ be set in the endpoint
configuration JSON (their keys will be omitted entirely).

This field is required, on the pretense that it is better to be
explicit about what needs filled in.

#### default

The literal default value that the plugin will infer if the field
is not specified (or is given as an empty value like "", or 0).

#### placeholder

Placeholder text that can be used in (at least) the Web UI HTML
forms for configuring this plugin.  It should either convey the
default value, or explain what the default behavior is.  For
example, if the S3 plugin's default `s3_host` parameter is
"s3.amazonaws.com", that would be a suitable placeholder.  On the
other hand, the PostgreSQL plugin defaults to backing up all
database unless `pg_database` is set, so a placeholder of "(all
databases)" (note the parentheses) would be appropriate.

This field is not required; if not present, there will be no
placeholder text.  Required fields with straightforward semantics,
like a _Username_ field, do not need placeholder text.

#### enum

A list of values that are allowed for fields of type `enum`.  If
the type of this field is `enum`, this key is **required**.
Otherrwise, this key is **forbidden**, and must not appear.

#### help

A helpful couple of sentences that explains what the field is used
for, what one might want to put in it, what format is expected,
etc.

#### example

A string that (in English) list possible values that could be
input _verbatim_, to assist operators in determining things like
proper formatting, whether to include the `https://` in URLs, etc.

### Web UI Form Field Display Example

Here is an example of a field, as displayed in the Web UI, as an
element of an HTML form:

![Form Field Example](field.png)

and here is an example of a field with an error:

![Form Field Error Example](field-error.png)

### CLI Form Field Display Example

Here is an example of a field, as displayed in the Web UID:

    Label
      (Hints) Lorem ipsum dolor sit amet, consectetur adipisciing
      elit, sed diam nonummy nibh euismod tincidunt ut laoreet dolore

      For example: example 1, example 2, or example 3

    Label:
    Placeholder> _

(this is just a mockup, I'm not entirely convinced that all the
 extra text is justified / justifiable)
