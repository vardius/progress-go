🗃️ progress-go
================
[![Build Status](https://travis-ci.org/vardius/progress-go.svg?branch=master)](https://travis-ci.org/vardius/progress-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/vardius/progress-go)](https://goreportcard.com/report/github.com/vardius/progress-go)
[![codecov](https://codecov.io/gh/vardius/progress-go/branch/master/graph/badge.svg)](https://codecov.io/gh/vardius/progress-go)
[![](https://godoc.org/github.com/vardius/progress-go?status.svg)](https://pkg.go.dev/github.com/vardius/progress-go)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/vardius/progress-go/blob/master/LICENSE.md)

<img align="right" height="180px" src="https://github.com/vardius/gorouter/blob/master/website/src/static/img/logo.png?raw=true" alt="logo" />

Go simple progress bar writing to output 

📖 ABOUT
==================================================
Contributors:

* [Rafał Lorenz](http://rafallorenz.com)

Want to contribute ? Feel free to send pull requests!

Have problems, bugs, feature ideas?
We are using the github [issue tracker](https://github.com/vardius/progress-go/issues) to manage them.

## 📚 Documentation

For __examples__ **visit [godoc#pkg-examples](http://godoc.org/github.com/vardius/progress-go#pkg-examples)**

For **GoDoc** reference, **visit [pkg.go.dev](https://pkg.go.dev/github.com/vardius/progress-go)**

🚏 HOW TO USE
==================================================

<img align="center" src="https://github.com/vardius/progress-go/blob/master/.github/demo.gif?raw=true" alt="Progress Bar CLI" />

## 🏫 Basic example
```go
package main

import (
	"log"

	"github.com/vardius/progress-go"
)

func main() {
	bar := progress.New(0, 10)

	_, _ = bar.Start()
	defer func() {
		if _, err := bar.Stop(); err != nil {
			log.Printf("faile to finish progress: %v", err)
		}
	}()

	for i := 0; i < 10; i++ {
		_, _ = bar.Advance(1)
	}
}
```

📜 [License](LICENSE.md)
-------

This package is released under the MIT license. See the complete license in the package.
