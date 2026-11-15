// Package id は主キーの文字列を作ります。データベースの自動採番は使いません。
package id

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// New はアプリが採番する主キーです。データベースの自動採番は使いません。
func New() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
