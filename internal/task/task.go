// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package task implements tasks involved in making a Go release.
package task

// CommunicationTasks combines communication tasks together.
type CommunicationTasks struct {
	AnnounceMailTasks
	TweetTasks
}
