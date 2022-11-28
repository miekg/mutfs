// Copyright 2020 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.

// Adapted by Miek Gieben to become mutfs.

// This is main program driver for a loopback filesystem that disallows destructive actions.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	flag "github.com/spf13/pflag"
)

// Mutfs is a loopback FS node disallowing destructive actions. Within a user defined grace period actions _are_
// allowed.
type MutNode struct {
	fs.LoopbackNode
}

var (
	Log   bool
	Grace time.Duration
)

var (
	_ = (fs.NodeOpener)((*MutNode)(nil))
	_ = (fs.NodeUnlinker)((*MutNode)(nil))
	_ = (fs.NodeRenamer)((*MutNode)(nil))
	_ = (fs.NodeSetxattrer)((*MutNode)(nil))
	_ = (fs.NodeSetattrer)((*MutNode)(nil))
	_ = (fs.NodeRmdirer)((*MutNode)(nil))
	_ = (fs.NodeRemovexattrer)((*MutNode)(nil))
)

func (n *MutNode) deny(ctx context.Context, name string) syscall.Errno {
	actualPath := filepath.Join(n.LoopbackNode.RootData.Path, filepath.Join(n.LoopbackNode.Path(n.LoopbackNode.Root()), name))
	caller, _ := fuse.FromContext(ctx)
	if bt, err := btime(actualPath); err == nil { // on success
		if since := time.Since(bt); since < Grace {
			if !Log {
				return fs.OK
			}
			log.Printf("Access granted to %q because of grace: %s, from pid %d and %d%d", actualPath, Grace-since, caller.Pid, caller.Owner.Uid, caller.Owner.Gid)
			return fs.OK
		}
	}

	if !Log {
		return syscall.EACCES
	}
	log.Printf("Write access denied to %q from pid %d, from %d/%d", actualPath, caller.Pid, caller.Owner.Uid, caller.Owner.Gid)
	return syscall.EACCES
}

func (n *MutNode) Unlink(ctx context.Context, name string) syscall.Errno {
	errno := n.deny(ctx, name)
	if errno != fs.OK {
		return errno
	}
	return n.LoopbackNode.Unlink(ctx, name)
}

func (n *MutNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	errno := n.deny(ctx, name)
	if errno != fs.OK {
		return errno
	}
	return n.LoopbackNode.Rmdir(ctx, name)
}

func (n *MutNode) Removexattr(ctx context.Context, attr string) syscall.Errno {
	errno := n.deny(ctx, "")
	if errno != fs.OK {
		return errno
	}
	return n.LoopbackNode.Removexattr(ctx, attr)
}

func (n *MutNode) Setxattr(ctx context.Context, attr string, data []byte, flags uint32) syscall.Errno {
	errno := n.deny(ctx, "")
	if errno != fs.OK {
		return errno
	}
	return n.LoopbackNode.Setxattr(ctx, attr, data, flags)
}

func (n *MutNode) Setattr(ctx context.Context, f fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	errno := n.deny(ctx, "")
	if errno != fs.OK {
		return errno
	}
	return n.LoopbackNode.Setattr(ctx, f, in, out)
}

func (n *MutNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	errno := n.deny(ctx, "")
	if errno != fs.OK {
		return errno
	}

	return n.LoopbackNode.Rename(ctx, name, newParent, newName, flags)
}

func (n *MutNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	if flags&syscall.O_CREAT != 0 {
		fs1, flags1, errno1 := n.LoopbackNode.Open(ctx, flags)
		if errno1 == syscall.ENOENT {
			return fs1, flags1, errno1
		}
	}

	// Only allow read access.
	switch {
	case flags&syscall.O_APPEND != 0:
		fallthrough
	case flags&syscall.O_WRONLY != 0:
		fallthrough
	case flags&syscall.O_TRUNC != 0:
		fallthrough
	case flags&syscall.O_RDWR != 0:
		errno := n.deny(ctx, "")
		if errno != fs.OK {
			return nil, 0, errno
		}
		return n.LoopbackNode.Open(ctx, flags)
	}

	// I don't know what 0x8000 is, syscall.O_* doesn't have such a value...
	flags = flags &^ 0x8000

	if flags == syscall.O_RDONLY {
		return n.LoopbackNode.Open(ctx, flags)
	}

	return nil, 0, syscall.EACCES
}

func New(rootData *fs.LoopbackRoot, _ *fs.Inode, _ string, _ *syscall.Stat_t) fs.InodeEmbedder {
	return &MutNode{LoopbackNode: fs.LoopbackNode{RootData: rootData}}
}

var flagOpts *[]string

func main() {
	flagOpts = flag.StringSliceP("opt", "o", nil, "options [debug,null,allow_other,ro,log,grace=<duration>")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Printf("usage: %s oldir newdir\n", path.Base(os.Args[0]))
		fmt.Printf("\noptions:\n")
		flag.PrintDefaults()
		os.Exit(2)
	}

	olddir := flag.Arg(0)
	for _, d := range []string{olddir, flag.Arg(1)} {
		fi, err := os.Stat(d)
		if err != nil {
			log.Fatalf("Can't stat %q: %s", d, err)
		}
		if !fi.IsDir() {
			log.Fatalf("%q isn't a directory", d)
		}
	}

	rootData := &fs.LoopbackRoot{
		NewNode: New,
		Path:    olddir,
	}
	mutnode := New(rootData, nil, "", nil)

	sec := time.Second
	opts := &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	}

	for _, o := range *flagOpts {
		switch {
		case o == "debug":
			opts.Debug = true
		case o == "null":
			opts.NullPermissions = true
		case o == "allow_other":
			opts.AllowOther = true
			opts.MountOptions.Options = append(opts.MountOptions.Options, "default_permissions")
		case o == "ro":
			opts.MountOptions.Options = append(opts.MountOptions.Options, "ro")
		case o == "log":
			Log = true
		case strings.HasPrefix(o, "grace="):
			xs := strings.Split(o, "=")
			if len(xs) != 2 {
				log.Fatalf("Wrongly specified grace: %s", o)
			}
			d, err := time.ParseDuration(xs[1])
			if err != nil {
				log.Fatalf("Wrongly specified grace: %s: %s", o, err)
			}
			Grace = d

		}
	}
	opts.MountOptions.Options = append(opts.MountOptions.Options, "fsname="+olddir)
	opts.MountOptions.Name = "mutfs"

	log.SetFlags(log.Lmicroseconds)
	server, err := fs.Mount(flag.Arg(1), mutnode, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	server.Wait()
}
