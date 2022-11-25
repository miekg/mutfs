%%%
title = "mutfs 5"
area = "File Formats Manual"
workgroup = "Mutfs Filesystem"
%%%

mutfs
=====

## Name

mutfs - immutable file system

## Synopsis

`mutfs [OPTION]...` *olddir* *newdir*

## Description

Mutfs is used as an overlay file system to make it immutable, write actions are only allowed when
creating a new file. Once things exists, they can't be changed or deleted.

Where options is a comma seperated list, currently supported:

* `debug`: enable debug logging.
* `null`: change *null* permissions to 0644 (files), 0755 (dirs).
* `log`: enable logging when a destructive action is tried.

Using `mount -t mutfs ~ /tmp/mut -o debug` will use mutfs (*if* the executable (`mount.mutfs`) can
be found in the path) to mount `~` under `/tmp`. Note that this "hangs" for as long the mount point
is mounted. Use the mutfs.sh shell script to make mutfs background.

Or you can install the following systemd mount unit:

~~~ ini
[Unit]
Description=Immutable Filesystem
After=network.target

[Mount]
What=<olddir>
Where=<newdir>
Type=mutfs
Options=debug

[Install]
WantedBy=multi-user.target
~~~

## Install

Copy mutfs and mount.mutfs to /usr/sbin. And potentially add a line to /etc/fstab;

~~~ fstab
/home/miek    /tmp/mut         mutfs     log,nouser   0 0
~~~

And adjust as needed.

## Examples

For example mount your home directory on a directory in `/tmp`: `/mutfs ~ /tmp/mut`, then you can
create the file `a`, but we can't update it after it's creation:

~~~ sh
% cd /tmp/mut
% echo 1 > a
% cat a
1
% echo 2 > a
zsh: permission denied: a
% rm a
rm: cannot remove 'a': Permission denied
~~~

## Author

Miek Gieben <miek@miek.nl>.
