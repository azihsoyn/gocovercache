package main

import (
	"testing"

	"github.com/kr/pretty"
)

func TestCalcCheckSum(t *testing.T) {
	checksum := calcCheckSum("./")
	pretty.Println("checksum : ", checksum)
}

func TestGetAbsolutePackageDir(t *testing.T) {
	pkgDir := getAbsolutePackageDir("github.com/azihsoyn/gocovercache")
	pretty.Println("pkgDir : ", pkgDir)
}
