// Copyright 2020 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted by Miek Gieben to become mutfs.

// This is main program driver for a loopback filesystem that disallows unlinks.
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
)

// Mutfs is a loopback FS node disallowing unlinks.
type MutNode struct {
	fs.LoopbackNode
}

var (
	_ = (fs.NodeOpener)((*MutNode)(nil))
	_ = (fs.NodeUnlinker)((*MutNode)(nil))
	_ = (fs.NodeRenamer)((*MutNode)(nil))
)

func (n *MutNode) Unlink(_ context.Context, _ string) syscall.Errno { return syscall.EACCES }
func (n *MutNode) Rename(_ context.Context, _ string, _ fs.InodeEmbedder, _ string, _ uint32) syscall.Errno {
	return syscall.EACCES
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

	// I don't know what 0x8000 is syscall.O_* doesn't have such a value...
	flags = flags &^ 0x8000

	if flags == syscall.O_RDONLY {
		return n.LoopbackNode.Open(ctx, flags)
	}

	return nil, 0, syscall.EACCES
}

func New(rootData *fs.LoopbackRoot, _ *fs.Inode, _ string, _ *syscall.Stat_t) fs.InodeEmbedder {
	return &MutNode{fs.LoopbackNode{RootData: rootData}}
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	debug := flag.Bool("debug", false, "print debugging messages.")
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Printf("usage: %s MOUNTPOINT ORIGINAL\n", path.Base(os.Args[0]))
		fmt.Printf("\noptions:\n")
		flag.PrintDefaults()
		os.Exit(2)
	}

	orig := flag.Arg(1)
	rootData := &fs.LoopbackRoot{
		NewNode: New,
		Path:    orig,
	}

	sec := time.Second
	opts := &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
	}
	opts.Debug = *debug
	opts.MountOptions.Options = append(opts.MountOptions.Options, "fsname="+orig)
	opts.MountOptions.Name = "mutfs"
	opts.NullPermissions = true

	server, err := fs.Mount(flag.Arg(0), New(rootData, nil, "", nil), opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	fmt.Println("Mounted!")
	server.Wait()
}
