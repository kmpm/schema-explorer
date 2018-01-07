package sdv

import (
	"fmt"
	"log"
	"time"
)

// roughly 3 months from when this is released into the wild
var Expiry = time.Date(2018, time.March, 1, 0, 0, 0, 0, time.UTC)
var CopyrightYear = 2018

func Licensing() {
	if time.Now().After(Expiry) {
		log.Panicf("Expired trial, contact %s to obtain a license", About.Email)
	}
}

func LicenseText() string {
	return fmt.Sprintf("This pre-release software will expire on: %s, contact %s for a license.", Expiry, About.Email)
}

func CopyrightText() string {
	return fmt.Sprintf("Copyright 2015-%d Tim Abell", CopyrightYear)
}
