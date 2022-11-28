%%%
title = "mutfs 5"
area = "File Formats Manual"
workgroup = "Mutfs Filesystem"
%%%

mutfs
=====

## Name

mutfs - immutable file system, with write grace period

## Synopsis

`mutfs [OPTION]...` *olddir* *newdir*

## Description

Mutfs is used as an overlay file system to make it immutable, write actions are only allowed when
creating a new file (or within a user specific grace period). Once things exists, they can't be
changed or deleted. A use-case might be to protect an backed up archive from a ransomware attack.
The attack will still happen, but at least it can't delete the old files (nor the encrypted ones
once created).

Options are:

- `-o opt,...`, where `opt` can be:
   * `debug`: enable debug logging.
   * `null`: change *null* permissions to 0644 (files), 0755 (dirs).
   * `allow_other`: everyone can access the files.
   * `ro`: make fully read-only.
   * `log`: enable logging when a destructive action is tried.
   * `grace=`*duration*, given a Go syntax duration will allow write operations for *duration*.

Using `mount -t mutfs ~ /tmp/mut -o debug,grace=5s` will use mutfs (*if* the executable
(`mount.mutfs`) can be found in the path) to mount `~` under `/tmp`. For up to 5 seconds after
file/directory creation destructive actions are allowed.

Note the grace period works by getting the files creation time via the `statx` system call, which
the underlying filesystem should support.

Or you can install the following systemd mount unit:

~~~ ini
[Unit]
Description=Immutable Filesystem
After=network.target

[Mount]
What=<olddir>
Where=<newdir>
Type=mutfs
Options=debug,allow_other

[Install]
WantedBy=multi-user.target
~~~

## Install

Copy mutfs and mount.mutfs to /usr/sbin. And potentially add a line to /etc/fstab;

~~~ fstab
/home/miek    /tmp/mut         mutfs     log,nouser,allow_other   0 0
~~~

And adjust as needed.

Re-exporting the filesystem only works if `allow_other` is specifed, otherwise the SMB/NFS daemon
cannot access the filesystem because only the user mounting it has access. Note unless you edit
`/etc/fuse.conf` only root can create mounts with this option specified.

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

## See Also

fuse(8) stat(2)

## Author

Miek Gieben <miek@miek.nl>.
