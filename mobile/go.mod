module github.com/cpacia/openbazaar3.0/mobile

go 1.13

require (
	github.com/cpacia/openbazaar3.0 v0.0.0-20200430214407-82c5b1019214
	golang.org/x/tools v0.0.0-20200117012304-6edc0a871e69 // indirect
)

replace (
	github.com/Roasbeef/ltcutil => github.com/ltcsuite/ltcutil v0.0.0-20181217130922-17f3b04680b6
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.4-0.20200121170514-da442c51f155
	github.com/go-critic/go-critic => github.com/go-critic/go-critic v0.4.0
	github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.21.0
	github.com/lightninglabs/neutrino => github.com/lightninglabs/neutrino v0.11.0
)
