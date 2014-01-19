GoGoGadget
==========

A framework for communicating to Arduino boards and components using Go and [Firmata][firmata-home].

See the docs at [godoc.org][doc].


Project is currently on hold and has not been tested in ~6 months.

###Setup
1. Make sure your [GOPATH][gopath-doc] is properly setup.
2. Run:
   `go get -u github.com/ZachMassia/GoGoGadget`
3. Upload the Standard Firmata sketch included with the Arduino IDE to your board.

## Example
```go
board, err := gadget.New("/dev/ttyACMO")
if err != nil {
    log.Fatal(err)
}
// Make sure the the serial connection is closed properly on exit.
defer board.Close()

// Board is ready to use.
```

[firmata-home]: http://firmata.org/wiki/Main_Page
[doc]: http://godoc.org/github.com/ZachMassia/GoGoGadget
[gopath-doc]: http://golang.org/doc/code.html#GOPATH

