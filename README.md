mutfs
=====

Mutfs, *immutable* filesystem is as filesystem that disallows deletes, *anything* created can not be
deleted or changed. It's a loopback filesystem that gets overlayed on a normal POSIX filesystem.

A use case could be that you want to archive a bunch of files, but fear a ransomware attack. After
such an attack *mutfs* will present you with the (newly created) encrypted files and the old ones.
Note you can't delete these newly created files either unless you go to the original mount point and
perform the deletes.

Example
-------

Mount your homedirectory: `/mutfs ~ /tmp/mut$USER`

Then:

~~~ sh
% cd /tmp/mutmiek
% echo 1 > a
% cat a
1
% echo 2 > a
zsh: permission denied: a
% rm a
rm: cannot remove 'a': Permission denied
~~~~

Synopsis
--------

`mutfs -o <option> olddir newdir`

Where options is a comma seperated list, currently supported:

* `debug`: enable debug logging

fstab
------

Using `mount -t fuse.mutfs ~ /tmp/mutmiek -o debug` will use mutfs (*if* the executable can be found
in the path) to mount ~ under /tmp/mutmiek (and that dir needs to exist as well).

TODO
====

The -o option is done last if following the mount spec. We need different flag lib to make that
happen.
