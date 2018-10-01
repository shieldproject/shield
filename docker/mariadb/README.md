You could use the `world.sql.gz` to test a backup of it:

    $ gunzip < world.sql.gz | mysql -u root -h 0.0.0.0 -p
