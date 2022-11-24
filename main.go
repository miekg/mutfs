// Copyright 2020 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.

// Adapted by Miek Gieben to become mutfs.

// This is main program driver for a loopback filesystem that disallows destructive actions.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// Mutfs is a loopback FS node disallowing destructive actions.
type MutNode struct {
	fs.LoopbackNode
}

var (
	_ = (fs.NodeOpener)((*MutNode)(nil))
	_ = (fs.NodeUnlinker)((*MutNode)(nil))
	_ = (fs.NodeRenamer)((*MutNode)(nil))
)

func deny(ctx context.Context, name string) syscall.Errno {
	caller, ok := fuse.FromContext(ctx)
	if !ok {
		return syscall.EACCES
	}
	if name != "" {
		log.Printf("Write access denied to %q from pid %d, from %d/%d", name, caller.Pid, caller.Owner.Uid, caller.Owner.Gid)
	} else {
		log.Printf("Write access denied from pid %d, from %d/%d", caller.Pid, caller.Owner.Uid, caller.Owner.Gid)
	}
	return syscall.EACCES
}

func (n *MutNode) Unlink(ctx context.Context, name string) syscall.Errno   { return deny(ctx, name) }
func (n *MutNode) Rmdir(ctx context.Context, name string) syscall.Errno    { return deny(ctx, name) }
func (n *MutNode) Removexattr(ctx context.Context, _ string) syscall.Errno { return deny(ctx, "") }
func (n *MutNode) Setxattr(ctx context.Context, _ string, _ []byte) (uint32, syscall.Errno) {
	return 0, deny(ctx, "")
}

func (n *MutNode) Setattr(ctx context.Context, f fs.FileHandle, _ *fuse.SetAttrIn, _ *fuse.AttrOut) syscall.Errno {
	return deny(ctx, "")
}

func (n *MutNode) Rename(ctx context.Context, name string, _ fs.InodeEmbedder, _ string, _ uint32) syscall.Errno {
	return deny(ctx, name)
}

func (n *MutNode) Setlkw(ctx context.Context, _ fs.FileHandle, _ uint64, _ *fuse.FileLock, _ uint32) syscall.Errno {
	return deny(ctx, "")
}

func (n *MutNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	if flags&syscall.O_CREAT != 0 {
		// this is racy, need a lock in n?
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
		return nil, 0, syscall.EACCES
	}

	// I don't know what 0x8000 is, syscall.O_* doesn't have such a value...
	flags = flags &^ 0x8000

	if flags == syscall.O_RDONLY {
		return n.LoopbackNode.Open(ctx, flags)
	}

	return nil, 0, syscall.EACCES
}

func New(rootData *fs.LoopbackRoot, _ *fs.Inode, _ string, _ *syscall.Stat_t) fs.InodeEmbedder {
	return &MutNode{fs.LoopbackNode{RootData: rootData}}
}

var flagOpts sliceFlag

func main() {
	flag.Var(&flagOpts, "o", "mount options, comma seperated")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Printf("usage: %s oldir newdir\n", path.Base(os.Args[0]))
		fmt.Printf("\noptions:\n")
		flag.PrintDefaults()
		os.Exit(2)
	}

	olddir := flag.Arg(0)
	rootData := &fs.LoopbackRoot{
		NewNode: New,
		Path:    olddir,
	}

	sec := time.Second
	opts := &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	}
	for _, o := range *flagOpts.Data {
		switch o {
		case "debug":
			opts.Debug = true
		}
	}
	opts.MountOptions.Options = append(opts.MountOptions.Options, "fsname="+olddir)
	opts.MountOptions.Name = "mutfs"
	opts.NullPermissions = true

	log.SetFlags(log.Lmicroseconds)
	server, err := fs.Mount(flag.Arg(1), New(rootData, nil, "", nil), opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	server.Wait()
}
